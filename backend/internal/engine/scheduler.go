package engine

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"business-scraping-service/backend/internal/config"
	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"
)

// Scheduler queues a run of the saved keywords every interval. Its timing lives in scheduler_state,
// so restarts don't move or skip runs, and ClaimDueRun makes sure only one instance starts each run.
type Scheduler struct {
	mu             sync.RWMutex
	workerPool     *WorkerPool
	queue          *JobQueue
	state          *repository.SchedulerStateRepository
	settingsRepo   *repository.SettingsRepository
	keywordRepo    *repository.KeywordRepository
	cfg            *config.Config
	interval       time.Duration
	nextRunAt      time.Time
	lastRunAt      *time.Time
	autoEnabled    bool
	maxPagesPerRun int
	pageSize       int
	maxResults     int
	enrichContacts bool
	engine         string
	stopChan       chan struct{}
}

func NewScheduler(
	workerPool *WorkerPool,
	queue *JobQueue,
	state *repository.SchedulerStateRepository,
	settingsRepo *repository.SettingsRepository,
	keywordRepo *repository.KeywordRepository,
	cfg *config.Config,
) *Scheduler {
	s := &Scheduler{
		workerPool:     workerPool,
		queue:          queue,
		state:          state,
		settingsRepo:   settingsRepo,
		keywordRepo:    keywordRepo,
		cfg:            cfg,
		interval:       24 * time.Hour,
		autoEnabled:    false,
		maxPagesPerRun: 3,
		pageSize:       20,
		maxResults:     60,
		enrichContacts: true,
		engine:         "places",
		stopChan:       make(chan struct{}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if settingsRepo != nil {
		var saved ScheduleInfo
		if err := settingsRepo.Get(ctx, "bali_schedule", &saved); err == nil {
			if saved.IntervalHours > 0 {
				s.interval = time.Duration(saved.IntervalHours) * time.Hour
			}
			s.autoEnabled = saved.AutoEnabled
			if saved.MaxPagesPerRun > 0 {
				s.maxPagesPerRun = saved.MaxPagesPerRun
			}
			if saved.MaxResults > 0 {
				s.maxResults = saved.MaxResults
			}
			s.enrichContacts = saved.EnrichContacts
			if saved.Engine != "" {
				s.engine = saved.Engine
			}
		}
	}

	// Timing: use the persisted next run; on first start, schedule the next UTC midnight.
	if st, err := state.Get(ctx); err != nil {
		slog.Error("Failed to load scheduler state", "error", err)
	} else {
		s.lastRunAt = st.LastRunAt
		if st.NextRunAt != nil {
			s.nextRunAt = *st.NextRunAt
		}
	}
	if s.nextRunAt.IsZero() {
		now := time.Now().UTC()
		s.nextRunAt = time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
		if err := state.SetNextRun(ctx, s.nextRunAt); err != nil {
			slog.Error("Failed to save scheduler next run", "error", err)
		}
	}

	return s
}

func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-s.stopChan:
				ticker.Stop()
				return
			case now := <-ticker.C:
				s.checkAndTrigger(ctx, now.UTC())
			}
		}
	}()
}

func (s *Scheduler) checkAndTrigger(ctx context.Context, now time.Time) {
	// The database is the source of truth: another instance may have run or moved the schedule.
	s.refreshTiming(ctx)

	s.mu.RLock()
	enabled := s.autoEnabled
	due := !now.Before(s.nextRunAt)
	interval := s.interval
	maxPages, pageSize, enrich := s.maxPagesPerRun, s.pageSize, s.enrichContacts
	s.mu.RUnlock()

	if !enabled || !due {
		return
	}

	// While the quota is exhausted, wait: the run starts as soon as the quota resets.
	if until, paused := s.queue.Quota().PausedUntil(now); paused {
		slog.Debug("Scheduled run waiting for Places quota reset", "until", until)
		return
	}

	claimed, next, err := s.state.ClaimDueRun(ctx, now, interval)
	if err != nil {
		slog.Error("Failed to claim scheduled run", "error", err)
		return
	}
	s.refreshTiming(ctx)
	if !claimed {
		return // another instance started it, or the schedule moved
	}

	slog.Info("Starting scheduled keyword run", "time", now.Format(time.RFC3339), "next_run_at", next)
	s.queueKeywordRuns(ctx, nil, maxPages, pageSize, enrich)
}

// refreshTiming reloads next/last run from the database.
func (s *Scheduler) refreshTiming(ctx context.Context) {
	st, err := s.state.Get(ctx)
	if err != nil {
		slog.Error("Failed to load scheduler state", "error", err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastRunAt = st.LastRunAt
	if st.NextRunAt != nil {
		s.nextRunAt = *st.NextRunAt
	}
}

// queueKeywordRuns queues one run for each target keyword: the given queries (created if new), or
// the active keywords, or (if none is active) all keywords. Keywords that are already queued or
// running are skipped. Returns the jobs created.
func (s *Scheduler) queueKeywordRuns(ctx context.Context, queries []string, maxPages, pageSize int, enrich bool) []*model.ScrapeJob {
	if s.workerPool.KeywordRunner().APIKey() == "" {
		slog.Error("Cannot run scheduled scrape: Google Places API key is not configured")
		return nil
	}
	if maxPages <= 0 {
		maxPages = 3
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	var targets []*model.ScraperKeyword
	var cleanQueries []string
	for _, q := range queries {
		if t := strings.TrimSpace(q); t != "" {
			cleanQueries = append(cleanQueries, t)
		}
	}

	if len(cleanQueries) > 0 {
		for _, q := range cleanQueries {
			kw, err := s.keywordRepo.GetByKeyword(ctx, q)
			if err != nil || kw == nil {
				kw, err = s.keywordRepo.Create(ctx, q)
			}
			if err != nil || kw == nil {
				slog.Error("Failed to find or create keyword", "keyword", q, "error", err)
				continue
			}
			targets = append(targets, kw)
		}
	} else {
		kws, err := s.keywordRepo.GetActiveKeywords(ctx)
		if err == nil && len(kws) > 0 {
			targets = kws
		} else {
			// Existing behaviour: with no active keyword, run all of them.
			allKws, _, _ := s.keywordRepo.List(ctx, model.ListKeywordFilter{Page: 1, PageSize: 100})
			targets = allKws
		}
	}

	if len(targets) == 0 {
		slog.Info("No search keywords configured for auto scraper")
		return nil
	}

	var created []*model.ScrapeJob
	skipped := 0
	for _, kw := range targets {
		job, err := s.queue.EnqueueKeyword(ctx, kw, KeywordRunOptions{MaxPages: maxPages, PageSize: pageSize, Enrich: enrich}, "auto_scraper_keyword")
		if err != nil {
			skipped++
			if err != ErrKeywordBusy {
				slog.Error("Failed to queue keyword", "keyword", kw.Keyword, "error", err)
			}
			continue
		}
		created = append(created, job)
	}
	slog.Info("Queued keyword run", "queued", len(created), "skipped", skipped)
	return created
}

type ScheduleInfo struct {
	NextRunAt        time.Time  `json:"next_run_at"`
	CountdownSeconds int64      `json:"countdown_seconds"`
	LastRunAt        *time.Time `json:"last_run_at,omitempty"`
	IntervalHours    int        `json:"interval_hours"`
	AutoEnabled      bool       `json:"auto_enabled"`
	MaxPagesPerRun   int        `json:"max_pages_per_run"`
	PageSize         int        `json:"page_size"`
	MaxResults       int        `json:"max_results"`
	EnrichContacts   bool       `json:"enrich_contacts"`
	Engine           string     `json:"engine"`
	QuotaPausedUntil *time.Time `json:"quota_paused_until,omitempty"`
}

func (s *Scheduler) GetScheduleInfo() ScheduleInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().UTC()
	countdown := int64(s.nextRunAt.Sub(now).Seconds())
	if countdown < 0 {
		countdown = 0
	}

	maxPages := s.maxPagesPerRun
	if maxPages <= 0 {
		maxPages = 3
	}
	pageSize := s.pageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	return ScheduleInfo{
		NextRunAt:        s.nextRunAt,
		CountdownSeconds: countdown,
		LastRunAt:        s.lastRunAt,
		IntervalHours:    int(s.interval.Hours()),
		AutoEnabled:      s.autoEnabled,
		MaxPagesPerRun:   maxPages,
		PageSize:         pageSize,
		MaxResults:       s.maxResults,
		EnrichContacts:   s.enrichContacts,
		Engine:           s.engine,
		QuotaPausedUntil: s.quotaPausedUntil(now),
	}
}

func (s *Scheduler) quotaPausedUntil(now time.Time) *time.Time {
	if until, paused := s.queue.Quota().PausedUntil(now); paused {
		return &until
	}
	return nil
}

func (s *Scheduler) UpdateSchedule(
	intervalHours int,
	autoEnabled bool,
	targetQueries []string,
	targetLocations []string,
	targetCategories []string,
	onlyNewBusinesses bool,
	requireWebsite bool,
	websiteFilter string,
	recencyFilter string,
	maxReviewCountForNew int,
	maxPagesPerRun int,
	pageSize int,
	maxResults int,
	enrichContacts bool,
	engine string,
) ScheduleInfo {
	s.mu.Lock()
	defer s.mu.Unlock()

	if intervalHours <= 0 {
		intervalHours = 24
	}
	s.interval = time.Duration(intervalHours) * time.Hour
	s.autoEnabled = autoEnabled

	if maxPagesPerRun > 0 {
		s.maxPagesPerRun = maxPagesPerRun
	}
	if pageSize > 0 {
		s.pageSize = pageSize
	}
	if maxResults > 0 {
		s.maxResults = maxResults
	}
	s.enrichContacts = enrichContacts
	if engine != "" {
		s.engine = engine
	}

	now := time.Now().UTC()
	s.nextRunAt = now.Add(s.interval)
	if err := s.state.SetNextRun(context.Background(), s.nextRunAt); err != nil {
		slog.Error("Failed to save scheduler next run", "error", err)
	}

	info := ScheduleInfo{
		NextRunAt:        s.nextRunAt,
		CountdownSeconds: int64(s.interval.Seconds()),
		LastRunAt:        s.lastRunAt,
		IntervalHours:    int(s.interval.Hours()),
		AutoEnabled:      s.autoEnabled,
		MaxPagesPerRun:   s.maxPagesPerRun,
		PageSize:         s.pageSize,
		MaxResults:       s.maxResults,
		EnrichContacts:   s.enrichContacts,
		Engine:           s.engine,
	}

	if s.settingsRepo != nil {
		go func(saved ScheduleInfo) {
			_ = s.settingsRepo.Set(context.Background(), "bali_schedule", saved)
		}(info)
	}

	return info
}

func (s *Scheduler) TriggerNow(
	ctx context.Context,
	queries []string,
	locations []string,
	categories []string,
	maxResults int,
	enrich bool,
	engine string,
	onlyNew bool,
	requireWebsite bool,
	websiteFilter string,
	recencyFilter string,
	maxReviews int,
) (*model.ScrapeJob, error) {
	s.mu.Lock()
	now := time.Now().UTC()
	s.lastRunAt = &now
	s.nextRunAt = now.Add(s.interval)
	interval := s.interval
	maxPages := s.maxPagesPerRun
	pageSize := s.pageSize
	s.mu.Unlock()

	if err := s.state.MarkRunNow(ctx, now, interval); err != nil {
		slog.Error("Failed to save scheduler run time", "error", err)
	}
	if maxResults > 0 {
		maxPages = (maxResults + 19) / 20
	}

	jobs := s.queueKeywordRuns(ctx, queries, maxPages, pageSize, enrich)
	if len(jobs) > 0 {
		return jobs[0], nil
	}
	return nil, nil
}
