package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
	_ "time/tzdata" // quota reset time is computed in America/Los_Angeles; don't rely on the image having tzdata

	"github.com/google/uuid"

	"business-scraping-service/backend/internal/engine/gmaps"
	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"
)

// ---------------------------------------------------------------------------------------------
// Quota pause

// NextQuotaReset is when Google resets daily Places quotas: midnight Pacific Time.
func NextQuotaReset(now time.Time) time.Time {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		loc = time.FixedZone("PT", -8*3600)
	}
	local := now.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, loc).UTC()
}

// QuotaDay is the Google quota day (Pacific date) that t falls in, as a UTC midnight date.
func QuotaDay(t time.Time) time.Time {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		loc = time.FixedZone("PT", -8*3600)
	}
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC)
}

type quotaStateStore interface {
	SetQuotaPausedUntil(ctx context.Context, until *time.Time) error
}

// QuotaGuard remembers that the Places quota is used up, so nothing calls the API (and fails) until
// it resets. It is persisted, so a restart doesn't forget it.
type QuotaGuard struct {
	mu    sync.RWMutex
	until time.Time
	store quotaStateStore
}

func NewQuotaGuard(store quotaStateStore, pausedUntil *time.Time) *QuotaGuard {
	g := &QuotaGuard{store: store}
	if pausedUntil != nil {
		g.until = *pausedUntil
	}
	return g
}

// PausedUntil returns the end of the current pause, if one is active.
func (g *QuotaGuard) PausedUntil(now time.Time) (time.Time, bool) {
	if g == nil {
		return time.Time{}, false
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	if now.Before(g.until) {
		return g.until, true
	}
	return time.Time{}, false
}

// Pause starts a pause until the next quota reset and returns its end.
func (g *QuotaGuard) Pause(ctx context.Context, now time.Time) time.Time {
	until := NextQuotaReset(now)
	g.mu.Lock()
	g.until = until
	g.mu.Unlock()
	if g.store != nil {
		if err := g.store.SetQuotaPausedUntil(ctx, &until); err != nil {
			slog.Error("Failed to persist Places quota pause", "error", err)
		}
	}
	return until
}

// Clear ends the pause (for example after a new API key is saved).
func (g *QuotaGuard) Clear(ctx context.Context) {
	g.mu.Lock()
	g.until = time.Time{}
	g.mu.Unlock()
	if g.store != nil {
		if err := g.store.SetQuotaPausedUntil(ctx, nil); err != nil {
			slog.Error("Failed to clear Places quota pause", "error", err)
		}
	}
}

func quotaPausedMessage(until time.Time) string {
	return fmt.Sprintf("Google Places API daily quota exceeded; scraping is paused until %s (quota reset).", until.Format("2006-01-02 15:04 MST"))
}

// ---------------------------------------------------------------------------------------------
// Job queue

type QueueOptions struct {
	Workers           int
	PollInterval      time.Duration
	HeartbeatInterval time.Duration
	StaleAfter        time.Duration // a PROCESSING job without a heartbeat for this long is requeued
	MaxAttempts       int
	JobTimeout        time.Duration
	MaintenanceEvery  time.Duration
	EnrichSweepAfter  time.Duration // leads left unenriched this long are picked up by the sweeper
}

func (o *QueueOptions) defaults() {
	if o.Workers <= 0 {
		o.Workers = 5
	}
	if o.PollInterval <= 0 {
		o.PollInterval = 2 * time.Second
	}
	if o.HeartbeatInterval <= 0 {
		o.HeartbeatInterval = 15 * time.Second
	}
	if o.StaleAfter <= 0 {
		o.StaleAfter = 2 * time.Minute
	}
	if o.MaxAttempts <= 0 {
		o.MaxAttempts = 3
	}
	if o.JobTimeout <= 0 {
		o.JobTimeout = 30 * time.Minute
	}
	if o.MaintenanceEvery <= 0 {
		o.MaintenanceEvery = time.Minute
	}
	if o.EnrichSweepAfter <= 0 {
		o.EnrichSweepAfter = 15 * time.Minute
	}
}

// JobQueue runs jobs stored in scrape_jobs with a fixed number of workers. Jobs survive restarts:
// interrupted ones are requeued on shutdown, and ones whose process died are found by their stale
// heartbeat and requeued.
type JobQueue struct {
	jobs     *repository.JobRepository
	keywords *repository.KeywordRepository
	leads    *repository.BusinessRepository
	wp       *WorkerPool
	quota    *QuotaGuard
	opts     QueueOptions
	alerts   *Alerter

	wake       chan struct{}
	softCtx    context.Context // cancelled to stop claiming new jobs
	softCancel context.CancelFunc
	hardCtx    context.Context // cancelled to interrupt running jobs
	hardCancel context.CancelFunc
	wg         sync.WaitGroup
	maintWG    sync.WaitGroup
	started    bool
}

func NewJobQueue(jobs *repository.JobRepository, keywords *repository.KeywordRepository, leads *repository.BusinessRepository, wp *WorkerPool, quota *QuotaGuard, opts QueueOptions) *JobQueue {
	opts.defaults()
	q := &JobQueue{
		jobs:     jobs,
		keywords: keywords,
		leads:    leads,
		wp:       wp,
		quota:    quota,
		opts:     opts,
		wake:     make(chan struct{}, opts.Workers),
	}
	q.softCtx, q.softCancel = context.WithCancel(context.Background())
	q.hardCtx, q.hardCancel = context.WithCancel(context.Background())
	return q
}

func (q *JobQueue) Quota() *QuotaGuard { return q.quota }

// SetAlerter sets where operational alerts go.
func (q *JobQueue) SetAlerter(a *Alerter) { q.alerts = a }

// Alerts returns the alerter (may be nil).
func (q *JobQueue) Alerts() *Alerter { return q.alerts }

// Start recovers work left by a previous process and starts the workers.
func (q *JobQueue) Start() {
	q.started = true
	q.maintain(true)

	for i := 0; i < q.opts.Workers; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}

	q.maintWG.Add(1)
	go func() {
		defer q.maintWG.Done()
		t := time.NewTicker(q.opts.MaintenanceEvery)
		defer t.Stop()
		for {
			select {
			case <-q.softCtx.Done():
				return
			case <-t.C:
				q.maintain(false)
			}
		}
	}()
}

// Notify wakes an idle worker after a job was enqueued.
func (q *JobQueue) Notify() {
	select {
	case q.wake <- struct{}{}:
	default:
	}
}

// Shutdown stops claiming jobs, lets running jobs finish for up to grace, then interrupts them
// (they are requeued and resume after the restart).
func (q *JobQueue) Shutdown(grace time.Duration) {
	q.softCancel()
	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		q.maintWG.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("Job queue stopped: all running jobs finished")
		q.hardCancel()
		return
	case <-time.After(grace):
	}

	slog.Warn("Shutdown grace period over, interrupting running jobs (they will be requeued)")
	q.hardCancel()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		slog.Error("Jobs did not stop after interrupt; they will be recovered by heartbeat on next start")
	}
}

// maintain requeues jobs whose process died, fixes keywords stuck in "processing", and enriches
// leads whose enrichment was interrupted.
func (q *JobQueue) maintain(startup bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	staleAfter := q.opts.StaleAfter
	if startup {
		// On startup no job can be running in this process yet, but another instance might be:
		// still require a stale heartbeat, just a shorter one.
		staleAfter = 3 * q.opts.HeartbeatInterval
	}
	requeued, failed, err := q.jobs.RecoverStale(ctx, staleAfter, q.opts.MaxAttempts)
	if err != nil {
		slog.Error("Failed to recover stale jobs", "error", err)
	} else if requeued+failed > 0 {
		slog.Warn("Recovered jobs interrupted by a restart", "requeued", requeued, "failed", failed)
		q.Notify()
	}

	if n, err := q.keywords.RestoreOrphanedProcessing(ctx); err != nil {
		slog.Error("Failed to restore stuck keywords", "error", err)
	} else if n > 0 {
		slog.Warn("Restored keywords stuck in processing", "count", n)
	}

	if !startup && q.leads != nil {
		q.sweepEnrichment()
	}
}

func (q *JobQueue) sweepEnrichment() {
	ctx, cancel := context.WithTimeout(q.softCtx, 5*time.Minute)
	defer cancel()
	leads, err := q.leads.ListPendingEnrichment(ctx, q.opts.EnrichSweepAfter, 25)
	if err != nil {
		slog.Error("Failed to list leads pending enrichment", "error", err)
		return
	}
	if len(leads) == 0 {
		return
	}
	byJob := map[uuid.UUID][]*model.BusinessLead{}
	for _, l := range leads {
		if l.JobID != nil {
			byJob[*l.JobID] = append(byJob[*l.JobID], l)
		}
	}
	total := 0
	for jobID, ls := range byJob {
		total += q.wp.enricher.Enrich(ctx, jobID, ls)
	}
	slog.Info("Enriched leads left over from interrupted runs", "candidates", len(leads), "enriched", total)
}

func (q *JobQueue) worker(n int) {
	defer q.wg.Done()
	for {
		if q.softCtx.Err() != nil {
			return
		}

		claimCtx, cancel := context.WithTimeout(q.softCtx, 10*time.Second)
		job, err := q.jobs.ClaimNext(claimCtx)
		cancel()
		if err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("Failed to claim job", "worker", n, "error", err)
		}

		if job == nil {
			select {
			case <-q.softCtx.Done():
				return
			case <-q.wake:
			case <-time.After(q.opts.PollInterval):
			}
			continue
		}

		q.execute(job)
	}
}

// writeCtx is for bookkeeping writes that must happen even while shutting down.
func writeCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

func (q *JobQueue) execute(job *model.ScrapeJob) {
	ctx, cancel := context.WithTimeout(q.hardCtx, q.opts.JobTimeout)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic recovered while running job", "job_id", job.ID, "panic", r)
			wctx, wcancel := writeCtx()
			defer wcancel()
			_ = q.jobs.UpdateStatus(wctx, job.ID, model.JobStatusFailed, 0, 0, 1, fmt.Sprintf("internal error: %v", r))
			if job.KeywordID != nil {
				_, _ = q.keywords.RestoreOrphanedProcessing(wctx)
			}
		}
	}()

	// Keep the heartbeat fresh so other instances / the next start know this job is alive.
	hbDone := make(chan struct{})
	defer close(hbDone)
	go func() {
		t := time.NewTicker(q.opts.HeartbeatInterval)
		defer t.Stop()
		for {
			select {
			case <-hbDone:
				return
			case <-t.C:
				hctx, hcancel := writeCtx()
				if err := q.jobs.Heartbeat(hctx, job.ID); err != nil {
					slog.Error("Job heartbeat failed", "job_id", job.ID, "error", err)
				}
				hcancel()
			}
		}
	}()

	slog.Info("Running job", "job_id", job.ID, "kind", job.Kind, "query", job.Query, "attempt", job.Attempts)

	if until, paused := q.quota.PausedUntil(time.Now()); paused && usesPlaces(job) {
		q.failWithoutRunning(job, quotaPausedMessage(until))
		return
	}

	var runErr error
	var kwResult *KeywordRunResult
	failed := false
	switch job.Kind {
	case model.JobKindKeyword:
		kwResult, runErr = q.runKeywordJob(ctx, job)
		failed = runErr != nil && (kwResult == nil || kwResult.Status == "")
	default:
		var status model.JobStatus
		status, runErr = q.wp.processJob(ctx, job)
		failed = status == model.JobStatusFailed
	}

	interrupted := runErr != nil && q.hardCtx.Err() != nil
	switch {
	case runErr != nil && gmaps.IsQuotaError(runErr):
		q.onQuotaExhausted()
	case failed && runErr != nil && !interrupted:
		q.alerts.Notify("job_failed", fmt.Sprintf("Scrape of %q failed: %v", job.Query, runErr))
	case kwResult != nil && kwResult.Status == model.KeywordStatusNoResults:
		q.alerts.Notify("no_results", fmt.Sprintf("Keyword %q returned no places from Google; try a different term or type.", job.Query))
	}

	// Interrupted by shutdown (not by its own timeout): put it back so it runs after the restart.
	if interrupted {
		wctx, wcancel := writeCtx()
		defer wcancel()
		if err := q.jobs.Requeue(wctx, job.ID, "Interrupted by a restart; queued again"); err != nil {
			slog.Error("Failed to requeue interrupted job", "job_id", job.ID, "error", err)
		} else if job.KeywordID != nil {
			_ = q.keywords.SetStatus(wctx, *job.KeywordID, model.KeywordStatusProcessing)
		}
		slog.Warn("Job interrupted by shutdown and requeued", "job_id", job.ID)
	}
}

func usesPlaces(job *model.ScrapeJob) bool {
	if job.Kind == model.JobKindKeyword {
		return true
	}
	engine, _ := job.Metadata["engine"].(string)
	return engine != "web"
}

func (q *JobQueue) failWithoutRunning(job *model.ScrapeJob, msg string) {
	wctx, cancel := writeCtx()
	defer cancel()
	if err := q.jobs.UpdateStatus(wctx, job.ID, model.JobStatusFailed, 0, 0, 0, msg); err != nil {
		slog.Error("Failed to fail job", "job_id", job.ID, "error", err)
	}
	if job.KeywordID != nil {
		_, _ = q.keywords.RestoreOrphanedProcessing(wctx)
	}
}

func (q *JobQueue) runKeywordJob(ctx context.Context, job *model.ScrapeJob) (*KeywordRunResult, error) {
	if job.KeywordID == nil {
		q.failWithoutRunning(job, "Keyword job has no keyword")
		return nil, nil
	}
	kw, err := q.keywords.GetByID(ctx, *job.KeywordID)
	if err != nil || kw == nil {
		q.failWithoutRunning(job, "Keyword no longer exists")
		return nil, nil
	}
	opts := KeywordRunOptions{
		MaxPages: metaInt(job.Metadata, "max_pages", 1),
		PageSize: metaInt(job.Metadata, "page_size", 20),
		Enrich:   job.EnrichContacts,
	}
	return q.wp.KeywordRunner().Run(ctx, kw, job.ID, opts)
}

// onQuotaExhausted pauses Places scraping until the quota resets and cancels queued jobs that would
// only fail the same way.
func (q *JobQueue) onQuotaExhausted() {
	ctx, cancel := writeCtx()
	defer cancel()
	until := q.quota.Pause(ctx, time.Now())
	msg := "Skipped: " + quotaPausedMessage(until)
	n, err := q.jobs.CancelPendingPlacesJobs(ctx, msg)
	if err != nil {
		slog.Error("Failed to cancel queued jobs after quota error", "error", err)
	}
	if _, err := q.keywords.RestoreOrphanedProcessing(ctx); err != nil {
		slog.Error("Failed to restore keywords after quota error", "error", err)
	}
	slog.Warn("Places API quota exhausted: scraping paused", "until", until, "cancelled_jobs", n)
	q.alerts.Notify("quota", fmt.Sprintf("Google Places quota or daily budget reached. Scraping paused until %s; %d queued jobs cancelled.", until.Format("2006-01-02 15:04 MST"), n))
}

// ---------------------------------------------------------------------------------------------
// Enqueueing

var ErrKeywordBusy = errors.New("keyword is already queued or being scraped")

// EnqueueManual queues a manual (New Job form) search.
func (q *JobQueue) EnqueueManual(ctx context.Context, job *model.ScrapeJob) error {
	job.Kind = model.JobKindManual
	job.Status = model.JobStatusPending
	if err := q.jobs.Create(ctx, job); err != nil {
		return err
	}
	q.Notify()
	return nil
}

// EnqueueKeyword queues one run of a keyword and marks the keyword as processing.
func (q *JobQueue) EnqueueKeyword(ctx context.Context, kw *model.ScraperKeyword, opts KeywordRunOptions, source string) (*model.ScrapeJob, error) {
	busy, err := q.jobs.HasActiveKeywordJob(ctx, kw.ID)
	if err != nil {
		return nil, err
	}
	if busy {
		return nil, ErrKeywordBusy
	}
	if opts.MaxPages <= 0 {
		opts.MaxPages = 1
	}
	if opts.PageSize <= 0 || opts.PageSize > 20 {
		opts.PageSize = 20
	}

	_, startPage := StartPosition(kw)
	kwID := kw.ID
	category := "Keyword Scrape"
	if source == "auto_scraper_keyword" {
		category = "Local Business"
	}
	job := &model.ScrapeJob{
		Kind:           model.JobKindKeyword,
		KeywordID:      &kwID,
		Status:         model.JobStatusPending,
		Query:          kw.Keyword,
		Location:       "Google Maps",
		Category:       category,
		MaxResults:     opts.MaxPages * opts.PageSize,
		EnrichContacts: opts.Enrich,
		Metadata: map[string]any{
			"source":       source,
			"keyword_id":   kw.ID.String(),
			"current_page": startPage,
			"max_pages":    opts.MaxPages,
			"page_size":    opts.PageSize,
		},
	}
	if err := q.jobs.Create(ctx, job); err != nil {
		return nil, err
	}
	if err := q.keywords.SetStatus(ctx, kw.ID, model.KeywordStatusProcessing); err != nil {
		slog.Error("Failed to mark keyword processing", "keyword_id", kw.ID, "error", err)
	}
	q.Notify()
	return job, nil
}

func metaInt(meta map[string]any, key string, def int) int {
	switch v := meta[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return def
}
