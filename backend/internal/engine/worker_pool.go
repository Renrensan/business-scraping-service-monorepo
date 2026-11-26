package engine

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"business-scraping-service/backend/internal/config"
	"business-scraping-service/backend/internal/engine/enrichment"
	"business-scraping-service/backend/internal/engine/gmaps"
	"business-scraping-service/backend/internal/engine/proxy"
	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"
)

type WorkerPool struct {
	cfg          *config.Config
	jobRepo      *repository.JobRepository
	businessRepo *repository.BusinessRepository
	redisClient  *redis.Client
	rotator      *proxy.Rotator
	webClient    *gmaps.WebClient
	placesClient *gmaps.PlacesAPIClient
	crawler      *enrichment.Crawler
	jobSem       chan struct{}
	enrichSem    chan struct{}
}

func NewWorkerPool(
	cfg *config.Config,
	jobRepo *repository.JobRepository,
	businessRepo *repository.BusinessRepository,
	redisClient *redis.Client,
) *WorkerPool {
	rotator := proxy.NewRotator(cfg.Proxies)
	webClient := gmaps.NewWebClient(rotator, cfg.DefaultUserAgent)
	placesClient := gmaps.NewPlacesAPIClient(cfg.GooglePlacesAPIKey)
	crawler := enrichment.NewCrawler(rotator, cfg.DefaultUserAgent)

	maxScrapers := cfg.MaxConcurrentScrapers
	if maxScrapers <= 0 {
		maxScrapers = 5
	}
	maxEnrichers := cfg.MaxConcurrentEnrichers
	if maxEnrichers <= 0 {
		maxEnrichers = 10
	}

	return &WorkerPool{
		cfg:          cfg,
		jobRepo:      jobRepo,
		businessRepo: businessRepo,
		redisClient:  redisClient,
		rotator:      rotator,
		webClient:    webClient,
		placesClient: placesClient,
		crawler:      crawler,
		jobSem:       make(chan struct{}, maxScrapers),
		enrichSem:    make(chan struct{}, maxEnrichers),
	}
}

func (wp *WorkerPool) SetGoogleAPIKey(apiKey string) {
	wp.cfg.GooglePlacesAPIKey = apiKey
	wp.placesClient = gmaps.NewPlacesAPIClient(apiKey)
}

func (wp *WorkerPool) SetProxies(proxies []string) {
	wp.rotator.SetProxies(proxies)
}

func (wp *WorkerPool) SubmitJob(job *model.ScrapeJob) {
	go func() {
		wp.jobSem <- struct{}{}
		defer func() { <-wp.jobSem }()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		wp.processJob(ctx, job)
	}()
}

func (wp *WorkerPool) processJob(ctx context.Context, job *model.ScrapeJob) {
	slog.Info("Starting scrape job", "job_id", job.ID, "query", job.Query, "location", job.Location)
	_ = wp.jobRepo.UpdateStatus(ctx, job.ID, model.JobStatusProcessing, 0, 0, 0, "")

	onlyNew, _ := job.Metadata["only_new_businesses"].(bool)
	searchQuery := gmaps.BuildSearchQueryWithFilter(job.Query, job.Location, job.Category, onlyNew)

	var leads []*model.BusinessLead
	var err error

	// 1. Extraction Phase (Google Places API if key provided, else HTML scraper)
	apiKey := wp.cfg.GooglePlacesAPIKey

	engineChoice, _ := job.Metadata["engine"].(string)

	if engineChoice != "web" && apiKey != "" {
		slog.Info("Executing extraction via Google Places API (New)", "query", searchQuery, "job_id", job.ID)
		client := gmaps.NewPlacesAPIClient(apiKey)
		leads, err = client.SearchPlacesText(ctx, searchQuery, job.MaxResults)
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

	totalScraped := len(leads)
	totalEnriched := 0
	errorCount := 0

	// 2. Storage & Upsert Phase
	for _, lead := range leads {
		lead.JobID = &job.ID
		// Keep Google's category: never replace it with the search text, or every result of a
		// search (hotels returned for "petshop") would be filed under that search's category.
		if lead.Category == "" {
			lead.Category = "Local Business"
		}

		// Record which search found this lead, separately from its real categories.
		if job.Query != "" && !strings.EqualFold(job.Query, "businesses") {
			lead.ScrapeKeywords = append(lead.ScrapeKeywords, job.Query)
		}
		if job.Category != "" && !strings.EqualFold(job.Category, "all categories") && !strings.EqualFold(job.Category, "any business") && job.Category != job.Query {
			lead.ScrapeKeywords = append(lead.ScrapeKeywords, job.Category)
		}

		if _, err := wp.businessRepo.Upsert(ctx, lead); err != nil {
			slog.Error("Failed to upsert lead", "name", lead.Name, "error", err)
			errorCount++
		}
	}

	if totalScraped > 0 {
		_ = wp.jobRepo.IncrementProgress(ctx, job.ID, totalScraped, 0, errorCount)
	}

	// 3. Contact & Social Enrichment Phase (if enabled and website exists)
	if job.EnrichContacts && len(leads) > 0 {
		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, lead := range leads {
			if lead.Website == "" {
				continue
			}
			wg.Add(1)
			go func(l *model.BusinessLead) {
				defer wg.Done()
				wp.enrichSem <- struct{}{}
				defer func() { <-wp.enrichSem }()

				enrichResult, err := wp.enrichLead(ctx, l)
				if err == nil && enrichResult != nil {
					mu.Lock()
					totalEnriched++
					mu.Unlock()
					_ = wp.jobRepo.IncrementProgress(ctx, job.ID, 0, 1, 0)
				}
			}(lead)
		}
		wg.Wait()
	}

	// 4. Final Completion
	finalStatus := model.JobStatusCompleted
	errMsg := ""
	if err != nil && totalScraped == 0 {
		finalStatus = model.JobStatusFailed
		errMsg = err.Error()
	} else if totalScraped == 0 {
		finalStatus = model.JobStatusFailed
		errMsg = "No listings found matching query"
	}

	_ = wp.jobRepo.UpdateStatus(ctx, job.ID, finalStatus, totalScraped, totalEnriched, errorCount, errMsg)
	slog.Info("Finished scrape job", "job_id", job.ID, "status", finalStatus, "scraped", totalScraped, "enriched", totalEnriched)
}

func (wp *WorkerPool) enrichLead(ctx context.Context, lead *model.BusinessLead) (*enrichment.EnrichmentResult, error) {
	if lead.Website == "" {
		return nil, nil
	}

	res, err := wp.crawler.EnrichWebsite(ctx, lead.Website)
	if err != nil {
		_ = wp.businessRepo.UpdateEnrichment(ctx, lead.ID, nil, nil, "", "", "", "", "", "", model.EnrichmentFailed)
		return nil, err
	}

	_ = wp.businessRepo.UpdateEnrichment(
		ctx,
		lead.ID,
		res.Emails,
		res.Phones,
		res.Socials.Facebook,
		res.Socials.Instagram,
		res.Socials.LinkedIn,
		res.Socials.Twitter,
		res.Socials.YouTube,
		res.Socials.TikTok,
		model.EnrichmentSuccess,
	)

	return res, nil
}
