package engine

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"business-scraping-service/backend/internal/config"
	"business-scraping-service/backend/internal/engine/enrichment"
	"business-scraping-service/backend/internal/engine/gmaps"
	"business-scraping-service/backend/internal/engine/proxy"
	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"

	"github.com/redis/go-redis/v9"
)

type WorkerPool struct {
	cfg          *config.Config
	jobRepo      *repository.JobRepository
	businessRepo *repository.BusinessRepository
	redisClient  *redis.Client
	rotator      *proxy.Rotator
	webClient    *gmaps.WebClient
	enricher     *LeadEnricher
	runner       *KeywordRunner

	keyMu  sync.RWMutex
	apiKey string
	budget int // daily Places request budget, 0 = none

	usage *repository.UsageRepository

	// newPlaces builds the Places client; tests swap it for a fake.
	newPlaces func(apiKey string) PlacesSearcher
}

func NewWorkerPool(
	cfg *config.Config,
	jobRepo *repository.JobRepository,
	businessRepo *repository.BusinessRepository,
	keywordRepo *repository.KeywordRepository,
	areaRepo *repository.AreaRepository,
	usageRepo *repository.UsageRepository,
	redisClient *redis.Client,
) *WorkerPool {
	rotator := proxy.NewRotator(cfg.Proxies)
	webClient := gmaps.NewWebClient(rotator, cfg.DefaultUserAgent)
	crawler := enrichment.NewCrawler(rotator, cfg.DefaultUserAgent)

	wp := &WorkerPool{
		cfg:          cfg,
		jobRepo:      jobRepo,
		businessRepo: businessRepo,
		redisClient:  redisClient,
		rotator:      rotator,
		webClient:    webClient,
		apiKey:       cfg.GooglePlacesAPIKey,
		usage:        usageRepo,
	}
	wp.newPlaces = func(key string) PlacesSearcher {
		return wp.PlacesClient(key)
	}
	wp.enricher = NewLeadEnricher(crawler, businessRepo, jobRepo, cfg.MaxConcurrentEnrichers)
	var areas AreaStore
	if areaRepo != nil {
		areas = areaRepo
	}
	wp.runner = NewKeywordRunner(keywordRepo, areas, businessRepo, jobRepo, wp.enricher, wp.APIKey)
	wp.runner.SetPlacesFactory(func(key string) PlacesSearcher { return wp.newPlaces(key) })
	return wp
}

// PlacesClient returns a Places client that counts every request against the daily budget.
func (wp *WorkerPool) PlacesClient(key string) *gmaps.PlacesAPIClient {
	return gmaps.NewPlacesAPIClient(key).WithGate(wp.budgetGate)
}

// budgetGate counts a request for today's quota day and refuses it once the budget is used up.
// A database error lets the request through: the budget is a safety net, not a hard dependency.
func (wp *WorkerPool) budgetGate(ctx context.Context) error {
	if wp.usage == nil {
		return nil
	}
	budget := wp.PlacesBudget()
	ok, err := wp.usage.TryConsume(ctx, QuotaDay(time.Now()), budget)
	if err != nil {
		slog.Error("Failed to count Places request", "error", err)
		return nil
	}
	if !ok {
		return fmt.Errorf("%w: daily request budget of %d reached", gmaps.ErrQuotaExceeded, budget)
	}
	return nil
}

func (wp *WorkerPool) PlacesBudget() int {
	wp.keyMu.RLock()
	defer wp.keyMu.RUnlock()
	return wp.budget
}

func (wp *WorkerPool) SetPlacesBudget(n int) {
	if n < 0 {
		n = 0
	}
	wp.keyMu.Lock()
	wp.budget = n
	wp.keyMu.Unlock()
}

// PlacesRequestsToday is how many Places requests were made in the current quota day.
func (wp *WorkerPool) PlacesRequestsToday(ctx context.Context) (int, error) {
	if wp.usage == nil {
		return 0, nil
	}
	return wp.usage.Requests(ctx, QuotaDay(time.Now()))
}

// APIKey is the Places API key in effect (Settings page key, else env key).
func (wp *WorkerPool) APIKey() string {
	wp.keyMu.RLock()
	defer wp.keyMu.RUnlock()
	return wp.apiKey
}

func (wp *WorkerPool) SetGoogleAPIKey(apiKey string) {
	wp.keyMu.Lock()
	wp.apiKey = apiKey
	wp.keyMu.Unlock()
}

// KeywordRunner is the shared keyword pipeline used by the keyword service and the scheduler.
func (wp *WorkerPool) KeywordRunner() *KeywordRunner {
	return wp.runner
}

func (wp *WorkerPool) SetProxies(proxies []string) {
	wp.rotator.SetProxies(proxies)
}

// processJob runs a manual search job to completion and returns its final status and the fetch error, if any.
func (wp *WorkerPool) processJob(ctx context.Context, job *model.ScrapeJob) (model.JobStatus, error) {
	slog.Info("Starting scrape job", "job_id", job.ID, "query", job.Query, "location", job.Location)
	if err := wp.jobRepo.UpdateStatus(ctx, job.ID, model.JobStatusProcessing, 0, 0, 0, ""); err != nil {
		slog.Error("Failed to mark job processing", "job_id", job.ID, "error", err)
	}

	onlyNew, _ := job.Metadata["only_new_businesses"].(bool)
	searchQuery := gmaps.BuildSearchQueryWithFilter(job.Query, job.Location, job.Category, onlyNew)

	var leads []*model.BusinessLead
	var err error

	// 1. Extraction Phase (Google Places API if a key is set, else the experimental HTML scraper)
	apiKey := wp.APIKey()
	engineChoice, _ := job.Metadata["engine"].(string)

	if engineChoice != "web" && apiKey != "" {
		slog.Info("Executing extraction via Google Places API (New)", "query", searchQuery, "job_id", job.ID)
		leads, err = wp.newPlaces(apiKey).SearchPlacesText(ctx, searchQuery, job.MaxResults)
	} else {
		slog.Info("Executing extraction via Maps Web Scraper", "query", searchQuery, "job_id", job.ID)
		leads, err = wp.webClient.SearchPlaces(ctx, searchQuery, job.MaxResults)
	}

	if err != nil {
		slog.Error("Extraction phase failed", "job_id", job.ID, "error", err)
	}

	totalScrapedRaw := len(leads)
	slog.Info("Extracted listings before filter", "job_id", job.ID, "count", totalScrapedRaw)

	// 1.1 Recency / Business Age Filter
	recencyFilter, _ := job.Metadata["recency_filter"].(string)
	if recencyFilter == "only_new" || (recencyFilter == "" && onlyNew) {
		var filtered []*model.BusinessLead
		for _, lead := range leads {
			isNew := false
			for _, sub := range lead.Subcategories {
				if strings.EqualFold(sub, "New Business") {
					isNew = true
					break
				}
			}
			if isNew || lead.BusinessStatus == "OPENING_SOON" {
				filtered = append(filtered, lead)
			}
		}
		slog.Info("Applied newly opened filter", "job_id", job.ID, "raw_count", totalScrapedRaw, "verified_new_count", len(filtered))
		leads = filtered
	} else if recencyFilter == "established" {
		var filtered []*model.BusinessLead
		for _, lead := range leads {
			isNew := false
			for _, sub := range lead.Subcategories {
				if strings.EqualFold(sub, "New Business") {
					isNew = true
					break
				}
			}
			if !isNew && lead.BusinessStatus != "OPENING_SOON" {
				filtered = append(filtered, lead)
			}
		}
		slog.Info("Applied established filter", "job_id", job.ID, "raw_count", totalScrapedRaw, "established_count", len(filtered))
		leads = filtered
	}
	// Note: If recencyFilter == "any" (or empty without onlyNew), all places are kept.

	// 1.2 Website Presence Filter
	websiteFilter, _ := job.Metadata["website_filter"].(string)
	requireWebsite, _ := job.Metadata["require_website"].(bool)
	if websiteFilter == "require_website" || websiteFilter == "with_website" || (websiteFilter == "" && requireWebsite) {
		var filtered []*model.BusinessLead
		for _, lead := range leads {
			if strings.TrimSpace(lead.Website) != "" {
				filtered = append(filtered, lead)
			}
		}
		slog.Info("Applied require website filter", "job_id", job.ID, "before", len(leads), "with_website", len(filtered))
		leads = filtered
	} else if websiteFilter == "no_website" || websiteFilter == "without_website" {
		var filtered []*model.BusinessLead
		for _, lead := range leads {
			if strings.TrimSpace(lead.Website) == "" {
				filtered = append(filtered, lead)
			}
		}
		slog.Info("Applied no website filter", "job_id", job.ID, "before", len(leads), "without_website", len(filtered))
		leads = filtered
	}
	// Note: If websiteFilter == "any" (or empty without requireWebsite), all places are kept.

	// 2. Storage: tag leads with the search that found them, separately from their real categories.
	var tags []string
	if job.Query != "" && !strings.EqualFold(job.Query, "businesses") {
		tags = append(tags, job.Query)
	}
	if job.Category != "" && !strings.EqualFold(job.Category, "all categories") && !strings.EqualFold(job.Category, "any business") && job.Category != job.Query {
		tags = append(tags, job.Category)
	}
	ingest := IngestLeads(ctx, wp.businessRepo, job.ID, leads, tags...)
	if ingest.Saved > 0 || ingest.Failed > 0 {
		if perr := wp.jobRepo.IncrementProgress(ctx, job.ID, ingest.Saved, 0, ingest.Failed); perr != nil {
			slog.Error("Failed to update job progress", "job_id", job.ID, "error", perr)
		}
	}

	// 3. Contact & Social Enrichment Phase (waits, so the job finishes with its final counts)
	totalEnriched := 0
	if job.EnrichContacts {
		totalEnriched = wp.enricher.Enrich(ctx, job.ID, ingest.Upserted)
	}

	// 4. Final Completion
	finalStatus, errMsg := manualJobOutcome(len(leads), ingest, err)
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if uerr := wp.jobRepo.UpdateStatus(writeCtx, job.ID, finalStatus, ingest.Saved, totalEnriched, ingest.Failed, errMsg); uerr != nil {
		slog.Error("Failed to update job status", "job_id", job.ID, "error", uerr)
	}
	slog.Info("Finished scrape job", "job_id", job.ID, "status", finalStatus, "scraped", ingest.Saved, "enriched", totalEnriched)
	if finalStatus == model.JobStatusFailed && err == nil && ctx.Err() != nil {
		err = ctx.Err()
	}
	return finalStatus, err
}

// manualJobOutcome decides a manual job's final status and message.
func manualJobOutcome(found int, ingest IngestResult, fetchErr error) (model.JobStatus, string) {
	switch {
	case ingest.Saved == 0 && fetchErr != nil:
		if gmaps.IsQuotaError(fetchErr) {
			return model.JobStatusFailed, "Google Places API daily quota exceeded; try again after the quota resets. " + fetchErr.Error()
		}
		return model.JobStatusFailed, fetchErr.Error()
	case ingest.Saved == 0 && ingest.Failed > 0:
		return model.JobStatusFailed, fmt.Sprintf("Failed to save any of %d leads: %v", ingest.Failed, ingest.LastErr)
	case ingest.Saved == 0 && found > 0:
		return model.JobStatusFailed, "No listings matched the selected filters"
	case ingest.Saved == 0:
		return model.JobStatusFailed, "No listings found matching query"
	case fetchErr != nil:
		return model.JobStatusCompleted, "Stopped early, kept the results found so far: " + fetchErr.Error()
	case ingest.Failed > 0:
		return model.JobStatusCompleted, fmt.Sprintf("%d leads could not be saved: %v", ingest.Failed, ingest.LastErr)
	}
	return model.JobStatusCompleted, ""
}
