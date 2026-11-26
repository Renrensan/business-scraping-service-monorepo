package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"business-scraping-service/backend/internal/engine/enrichment"
	"business-scraping-service/backend/internal/engine/gmaps"
	"business-scraping-service/backend/internal/model"
)

// The scrape pipeline shared by every entry point (manual jobs, the keyword "Scrape" button and the
// scheduler): fetch from Places, upsert leads, update the keyword cursor and job, then enrich websites.
// The small interfaces below are what the pipeline needs from the repositories, so it can be tested
// without a database.

type PlacesSearcher interface {
	SearchPlacesText(ctx context.Context, query string, maxResults int) ([]*model.BusinessLead, error)
	SearchWithCursor(ctx context.Context, spec gmaps.SearchSpec, startPageToken string, maxPages int, pageSize int) ([]*model.BusinessLead, string, int, bool, error)
}

type LeadStore interface {
	Upsert(ctx context.Context, lead *model.BusinessLead) (bool, error)
	UpdateEnrichment(ctx context.Context, id uuid.UUID, emails, phones []string, fb, ig, li, tw, yt, tt string, status model.EnrichmentStatus) error
}

type JobStore interface {
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.JobStatus, totalScraped, totalEnriched, errorCount int, errMsg string) error
	IncrementProgress(ctx context.Context, id uuid.UUID, scrapedDelta, enrichedDelta, errorDelta int) error
}

type KeywordStore interface {
	UpdateCursor(ctx context.Context, id uuid.UUID, nextToken string, currentPage int, addLeads int, status model.KeywordStatus) error
	SetStatus(ctx context.Context, id uuid.UUID, status model.KeywordStatus) error
	ResetCursor(ctx context.Context, id uuid.UUID) error
	RecordRunOutcome(ctx context.Context, id uuid.UUID, newPlaces int, now time.Time) error

	// Area keyword cells
	CountCells(ctx context.Context, keywordID uuid.UUID) (int, error)
	StartCycle(ctx context.Context, keywordID uuid.UUID, area *model.Area) error
	NextPendingCell(ctx context.Context, keywordID uuid.UUID) (*model.KeywordCell, error)
	SaveCellProgress(ctx context.Context, c *model.KeywordCell) error
	SplitCell(ctx context.Context, c *model.KeywordCell) error
	CycleResults(ctx context.Context, keywordID uuid.UUID) (int, error)
}

type AreaStore interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Area, error)
}

type WebsiteEnricher interface {
	EnrichWebsite(ctx context.Context, website string) (*enrichment.EnrichmentResult, error)
}

// APIKeySource returns the Places API key currently in effect.
type APIKeySource func() string

// ---------------------------------------------------------------------------------------------
// Ingest

type IngestResult struct {
	Saved    int // leads written (new or merged)
	New      int // leads that did not exist before
	Failed   int // leads whose upsert failed
	LastErr  error
	Upserted []*model.BusinessLead // leads that were saved, with their database IDs set
}

// IngestLeads saves leads for a job and tags them with the searches that found them.
// Google's category is kept: the search text never replaces it.
func IngestLeads(ctx context.Context, store LeadStore, jobID uuid.UUID, leads []*model.BusinessLead, scrapeKeywords ...string) IngestResult {
	var res IngestResult
	for _, lead := range leads {
		lead.JobID = &jobID
		if strings.TrimSpace(lead.Category) == "" {
			lead.Category = "Local Business"
		}
		for _, kw := range scrapeKeywords {
			if kw = strings.TrimSpace(kw); kw != "" {
				lead.ScrapeKeywords = append(lead.ScrapeKeywords, kw)
			}
		}

		wasNew, err := store.Upsert(ctx, lead)
		if err != nil {
			slog.Error("Failed to upsert lead", "job_id", jobID, "name", lead.Name, "error", err)
			res.Failed++
			res.LastErr = err
			continue
		}
		res.Saved++
		if wasNew {
			res.New++
		}
		res.Upserted = append(res.Upserted, lead)
	}
	return res
}

// ---------------------------------------------------------------------------------------------
// Enrichment

// LeadEnricher crawls lead websites for emails, phones and social links. One instance is shared by
// all scrape paths, so the concurrency limit applies across all of them.
type LeadEnricher struct {
	crawler WebsiteEnricher
	leads   LeadStore
	jobs    JobStore
	sem     chan struct{}
	timeout time.Duration
}

func NewLeadEnricher(crawler WebsiteEnricher, leads LeadStore, jobs JobStore, concurrency int) *LeadEnricher {
	if concurrency <= 0 {
		concurrency = 10
	}
	return &LeadEnricher{
		crawler: crawler,
		leads:   leads,
		jobs:    jobs,
		sem:     make(chan struct{}, concurrency),
		timeout: 10 * time.Minute,
	}
}

// Enrich crawls the website of every lead that has one and returns how many succeeded.
// It blocks until all crawls finish.
func (e *LeadEnricher) Enrich(ctx context.Context, jobID uuid.UUID, leads []*model.BusinessLead) int {
	if e == nil || e.crawler == nil || len(leads) == 0 {
		return 0
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	enriched := 0

	for _, lead := range leads {
		if strings.TrimSpace(lead.Website) == "" || lead.EnrichmentStatus == model.EnrichmentSuccess || lead.ID == uuid.Nil {
			continue
		}

		wg.Add(1)
		go func(l *model.BusinessLead) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					slog.Error("panic recovered in enrichment", "lead", l.Name, "panic", r)
				}
			}()

			select {
			case e.sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-e.sem }()

			res, err := e.crawler.EnrichWebsite(ctx, l.Website)
			if err != nil || res == nil {
				if uerr := e.leads.UpdateEnrichment(ctx, l.ID, nil, nil, "", "", "", "", "", "", model.EnrichmentFailed); uerr != nil {
					slog.Error("Failed to save enrichment failure", "lead_id", l.ID, "error", uerr)
				}
				return
			}

			if uerr := e.leads.UpdateEnrichment(ctx, l.ID,
				res.Emails, res.Phones,
				res.Socials.Facebook, res.Socials.Instagram, res.Socials.LinkedIn,
				res.Socials.Twitter, res.Socials.YouTube, res.Socials.TikTok,
				model.EnrichmentSuccess,
			); uerr != nil {
				slog.Error("Failed to save enrichment", "lead_id", l.ID, "error", uerr)
				return
			}

			mu.Lock()
			enriched++
			mu.Unlock()
			if e.jobs != nil {
				if perr := e.jobs.IncrementProgress(ctx, jobID, 0, 1, 0); perr != nil {
					slog.Error("Failed to update job enrichment progress", "job_id", jobID, "error", perr)
				}
			}
		}(lead)
	}

	wg.Wait()
	return enriched
}

// EnrichInBackground runs Enrich with its own timeout, detached from the caller's context.
func (e *LeadEnricher) EnrichInBackground(jobID uuid.UUID, leads []*model.BusinessLead) {
	if e == nil || len(leads) == 0 {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
		defer cancel()
		e.Enrich(ctx, jobID, leads)
	}()
}

// ---------------------------------------------------------------------------------------------
// Keyword runs

type KeywordRunOptions struct {
	MaxPages int
	PageSize int
	Enrich   bool
}

type KeywordRunResult struct {
	Leads        int
	NewLeads     int
	FailedLeads  int
	PagesFetched int
	StartPage    int
	CurrentPage  int
	Status       model.KeywordStatus
}

// KeywordRunner runs one keyword's cursor-based scrape. Used by the keyword "Scrape" button and the scheduler.
type KeywordRunner struct {
	keywords  KeywordStore
	areas     AreaStore
	leads     LeadStore
	jobs      JobStore
	enricher  *LeadEnricher
	apiKey    APIKeySource
	newPlaces func(apiKey string) PlacesSearcher
}

func NewKeywordRunner(keywords KeywordStore, areas AreaStore, leads LeadStore, jobs JobStore, enricher *LeadEnricher, apiKey APIKeySource) *KeywordRunner {
	return &KeywordRunner{
		keywords: keywords,
		areas:    areas,
		leads:    leads,
		jobs:     jobs,
		enricher: enricher,
		apiKey:   apiKey,
		newPlaces: func(key string) PlacesSearcher {
			return gmaps.NewPlacesAPIClient(key)
		},
	}
}

// SetPlacesFactory swaps the Places client constructor (tests use a fake).
func (r *KeywordRunner) SetPlacesFactory(f func(apiKey string) PlacesSearcher) {
	r.newPlaces = f
}

func (r *KeywordRunner) APIKey() string {
	if r.apiKey == nil {
		return ""
	}
	return r.apiKey()
}

// StartPosition is where a run of this keyword begins: its saved cursor, or page 1 for a new cycle.
func StartPosition(kw *model.ScraperKeyword) (token string, page int) {
	if kw.NextPageToken == "" || kw.Status == model.KeywordStatusCompleted || kw.Status == model.KeywordStatusNoResults {
		return "", 1
	}
	page = kw.CurrentPage
	if page < 1 {
		page = 1
	}
	return kw.NextPageToken, page
}

var ErrNoAPIKey = errors.New("google places api key is not configured")

// Run scrapes one keyword for a job that already exists, and always leaves the job in a final state
// (COMPLETED or FAILED). On a fetch error the keyword's cursor is left untouched so the next run
// continues from the same page. Quota errors are returned as-is so callers can stop their whole run.
func (r *KeywordRunner) Run(ctx context.Context, kw *model.ScraperKeyword, jobID uuid.UUID, opts KeywordRunOptions) (*KeywordRunResult, error) {
	if opts.MaxPages <= 0 {
		opts.MaxPages = 1
	}
	if opts.PageSize <= 0 || opts.PageSize > 20 {
		opts.PageSize = 20
	}

	token, page := StartPosition(kw)
	result := &KeywordRunResult{StartPage: page, CurrentPage: page}

	restoreStatus := kw.Status
	if restoreStatus == model.KeywordStatusProcessing || restoreStatus == "" {
		restoreStatus = model.KeywordStatusReady
	}

	fail := func(err error, msg string) (*KeywordRunResult, error) {
		r.finishJob(ctx, jobID, model.JobStatusFailed, 0, 1, msg)
		r.setKeywordStatus(ctx, kw.ID, restoreStatus)
		return result, err
	}

	key := r.APIKey()
	if key == "" {
		return fail(ErrNoAPIKey, ErrNoAPIKey.Error())
	}
	places := r.newPlaces(key)

	if kw.Mode == model.KeywordModeArea {
		return r.runArea(ctx, kw, jobID, opts, places, fail)
	}

	spec := gmaps.SearchSpec{Query: kw.Keyword, IncludedType: kw.IncludedType}
	leads, nextToken, pagesFetched, cycleFinished, err := places.SearchWithCursor(ctx, spec, token, opts.MaxPages, opts.PageSize)
	if err != nil && gmaps.IsInvalidPageTokenError(err) && token != "" {
		slog.Warn("Page token expired or invalid, restarting keyword from page 1", "keyword", kw.Keyword, "error", err)
		if rerr := r.keywords.ResetCursor(ctx, kw.ID); rerr != nil {
			slog.Error("Failed to reset keyword cursor", "keyword_id", kw.ID, "error", rerr)
		}
		token, page = "", 1
		result.StartPage, result.CurrentPage = 1, 1
		restoreStatus = model.KeywordStatusReady
		leads, nextToken, pagesFetched, cycleFinished, err = places.SearchWithCursor(ctx, spec, "", opts.MaxPages, opts.PageSize)
	}

	if err != nil {
		if gmaps.IsQuotaError(err) {
			slog.Warn("Places API quota exhausted, keyword scrape paused", "keyword", kw.Keyword, "error", err)
			return fail(err, "Google Places API daily quota exceeded; try again after the quota resets. "+err.Error())
		}
		slog.Error("Google Places fetch failed for keyword", "keyword", kw.Keyword, "error", err)
		return fail(err, fmt.Sprintf("Places error: %v", err))
	}

	ingest := IngestLeads(ctx, r.leads, jobID, leads, kw.Keyword)
	result.Leads = len(leads)
	result.NewLeads = ingest.New
	result.FailedLeads = ingest.Failed
	result.PagesFetched = pagesFetched
	result.CurrentPage = page + pagesFetched

	if len(leads) > 0 && ingest.Saved == 0 {
		// Nothing could be saved: keep the cursor so these pages are fetched again next time.
		return fail(ingest.LastErr, fmt.Sprintf("Failed to save any of %d leads: %v", len(leads), ingest.LastErr))
	}

	status := model.NextKeywordStatus(token == "", len(leads), nextToken, cycleFinished)
	if status != model.KeywordStatusPaginating {
		nextToken = ""
	}
	result.Status = status

	if cerr := r.keywords.UpdateCursor(ctx, kw.ID, nextToken, result.CurrentPage, ingest.Saved, status); cerr != nil {
		slog.Error("Failed to update keyword cursor", "keyword_id", kw.ID, "error", cerr)
		return fail(cerr, fmt.Sprintf("Leads saved but the keyword cursor could not be updated: %v", cerr))
	}

	r.recordOutcome(ctx, kw.ID, ingest.New)

	msg := ""
	switch {
	case status == model.KeywordStatusNoResults:
		msg = "Google returned no places for this keyword; try a different wording"
		slog.Warn("Keyword returned no results", "keyword", kw.Keyword)
	case ingest.Failed > 0:
		msg = fmt.Sprintf("%d of %d leads could not be saved: %v", ingest.Failed, len(leads), ingest.LastErr)
	}
	r.finishJob(ctx, jobID, model.JobStatusCompleted, ingest.Saved, ingest.Failed, msg)

	if opts.Enrich {
		r.enricher.EnrichInBackground(jobID, ingest.Upserted)
	}

	return result, nil
}

func (r *KeywordRunner) recordOutcome(ctx context.Context, id uuid.UUID, newPlaces int) {
	if err := r.keywords.RecordRunOutcome(ctx, id, newPlaces, time.Now().UTC()); err != nil {
		slog.Error("Failed to record keyword run outcome", "keyword_id", id, "error", err)
	}
}

// Area keywords: each cycle starts with one cell covering the area. A cell is searched with the
// area rectangle as a hard limit; when it returns Google's maximum (60) it is split into four
// quarters, down to maxCellDepth, so busy areas are covered beyond 60 places.
const (
	maxCellDepth    = 3
	maxPagesPerCell = 3
	placesPerCell   = 60
)

func (r *KeywordRunner) runArea(ctx context.Context, kw *model.ScraperKeyword, jobID uuid.UUID, opts KeywordRunOptions, places PlacesSearcher,
	fail func(error, string) (*KeywordRunResult, error)) (*KeywordRunResult, error) {

	result := &KeywordRunResult{StartPage: kw.CurrentPage}
	if kw.AreaID == nil || r.areas == nil {
		return fail(errors.New("area keyword has no area"), "This keyword's area was removed")
	}
	area, err := r.areas.GetByID(ctx, *kw.AreaID)
	if err != nil || area == nil {
		return fail(fmt.Errorf("area not found: %v", err), "This keyword's area was removed")
	}

	cells, err := r.keywords.CountCells(ctx, kw.ID)
	if err != nil {
		return fail(err, fmt.Sprintf("Failed to load area progress: %v", err))
	}
	newCycle := cells == 0 || kw.Status == model.KeywordStatusCompleted || kw.Status == model.KeywordStatusNoResults
	if newCycle {
		if err := r.keywords.StartCycle(ctx, kw.ID, area); err != nil {
			return fail(err, fmt.Sprintf("Failed to start area cycle: %v", err))
		}
		result.StartPage = 1
	}

	var (
		pagesLeft = opts.MaxPages
		pagesDone int
		totalSeen int
		saved     int
		newLeads  int
		failed    int
		lastErr   error
		upserted  []*model.BusinessLead
		stopErr   error
	)

	for pagesLeft > 0 {
		cell, err := r.keywords.NextPendingCell(ctx, kw.ID)
		if err != nil {
			stopErr = err
			break
		}
		if cell == nil {
			break // every cell done: cycle complete
		}

		spec := gmaps.SearchSpec{
			Query:        kw.Term,
			IncludedType: kw.IncludedType,
			Rect: &gmaps.Rect{
				Low:  gmaps.LatLng{Latitude: cell.LowLat, Longitude: cell.LowLng},
				High: gmaps.LatLng{Latitude: cell.HighLat, Longitude: cell.HighLng},
			},
		}
		want := maxPagesPerCell - cell.PagesFetched
		if want > pagesLeft {
			want = pagesLeft
		}
		if want <= 0 {
			want = 1
		}

		leads, next, fetched, finished, err := places.SearchWithCursor(ctx, spec, cell.NextPageToken, want, opts.PageSize)
		if err != nil && gmaps.IsInvalidPageTokenError(err) && cell.NextPageToken != "" {
			slog.Warn("Cell page token expired, restarting cell", "keyword", kw.Keyword, "cell", cell.ID)
			cell.NextPageToken, cell.PagesFetched, cell.Results = "", 0, 0
			leads, next, fetched, finished, err = places.SearchWithCursor(ctx, spec, "", want, opts.PageSize)
		}
		if err != nil {
			if pagesDone == 0 {
				if gmaps.IsQuotaError(err) {
					return fail(err, "Google Places API daily quota exceeded; try again after the quota resets. "+err.Error())
				}
				return fail(err, fmt.Sprintf("Places error: %v", err))
			}
			stopErr = err // keep what earlier cells found
			break
		}

		ingest := IngestLeads(ctx, r.leads, jobID, leads, kw.Keyword)
		if len(leads) > 0 && ingest.Saved == 0 {
			return fail(ingest.LastErr, fmt.Sprintf("Failed to save any of %d leads: %v", len(leads), ingest.LastErr))
		}
		saved += ingest.Saved
		newLeads += ingest.New
		failed += ingest.Failed
		if ingest.LastErr != nil {
			lastErr = ingest.LastErr
		}
		upserted = append(upserted, ingest.Upserted...)
		totalSeen += len(leads)

		cell.PagesFetched += fetched
		cell.Results += len(leads)
		pagesLeft -= fetched
		pagesDone += fetched

		if finished || next == "" {
			if cell.Results >= placesPerCell && cell.Depth < maxCellDepth {
				err = r.keywords.SplitCell(ctx, cell)
			} else {
				cell.Status, cell.NextPageToken = "done", ""
				err = r.keywords.SaveCellProgress(ctx, cell)
			}
		} else {
			cell.NextPageToken = next
			err = r.keywords.SaveCellProgress(ctx, cell)
		}
		if err != nil {
			return fail(err, fmt.Sprintf("Leads saved but area progress could not be updated: %v", err))
		}
		if fetched == 0 {
			break // defensive: no progress
		}
	}

	pending, err := r.keywords.NextPendingCell(ctx, kw.ID)
	if err != nil {
		return fail(err, fmt.Sprintf("Failed to load area progress: %v", err))
	}
	status := model.KeywordStatusPaginating
	if pending == nil {
		status = model.KeywordStatusCompleted
		if n, err := r.keywords.CycleResults(ctx, kw.ID); err == nil && n == 0 {
			status = model.KeywordStatusNoResults
		}
	}

	page := kw.CurrentPage
	if newCycle || page < 1 {
		page = 1
	}
	result.Leads = totalSeen
	result.NewLeads = newLeads
	result.FailedLeads = failed
	result.PagesFetched = pagesDone
	result.CurrentPage = page + pagesDone
	result.Status = status

	if err := r.keywords.UpdateCursor(ctx, kw.ID, "", result.CurrentPage, saved, status); err != nil {
		return fail(err, fmt.Sprintf("Leads saved but the keyword could not be updated: %v", err))
	}
	r.recordOutcome(ctx, kw.ID, newLeads)

	msg := ""
	switch {
	case stopErr != nil:
		msg = "Stopped early, kept the results found so far: " + stopErr.Error()
	case status == model.KeywordStatusNoResults:
		msg = "Google returned no places for this term in this area; try a different term or type"
	case failed > 0:
		msg = fmt.Sprintf("%d of %d leads could not be saved: %v", failed, totalSeen, lastErr)
	}
	r.finishJob(ctx, jobID, model.JobStatusCompleted, saved, failed, msg)

	if opts.Enrich {
		r.enricher.EnrichInBackground(jobID, upserted)
	}
	return result, stopErr
}

func (r *KeywordRunner) finishJob(ctx context.Context, jobID uuid.UUID, status model.JobStatus, scraped, errCount int, msg string) {
	if r.jobs == nil {
		return
	}
	// Use a fresh context so a cancelled run still records its final state.
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if err := r.jobs.UpdateStatus(writeCtx, jobID, status, scraped, 0, errCount, msg); err != nil {
		slog.Error("Failed to update job status", "job_id", jobID, "status", status, "error", err)
	}
}

func (r *KeywordRunner) setKeywordStatus(ctx context.Context, id uuid.UUID, status model.KeywordStatus) {
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if err := r.keywords.SetStatus(writeCtx, id, status); err != nil {
		slog.Error("Failed to set keyword status", "keyword_id", id, "status", status, "error", err)
	}
}
