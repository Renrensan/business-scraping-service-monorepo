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
	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"
)

type Scheduler struct {
	mu             sync.RWMutex
	workerPool     *WorkerPool
	jobRepo        *repository.JobRepository
	businessRepo   *repository.BusinessRepository
	settingsRepo   *repository.SettingsRepository
	keywordRepo    *repository.KeywordRepository
	cfg            *config.Config
	crawler        *enrichment.Crawler
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
	jobRepo *repository.JobRepository,
	businessRepo *repository.BusinessRepository,
	settingsRepo *repository.SettingsRepository,
	keywordRepo *repository.KeywordRepository,
	cfg *config.Config,
) *Scheduler {
	now := time.Now().UTC()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	if nextMidnight.Sub(now) < 1*time.Hour {
		nextMidnight = nextMidnight.Add(24 * time.Hour)
	}

	userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"
	if cfg != nil && cfg.DefaultUserAgent != "" {
		userAgent = cfg.DefaultUserAgent
	}

	s := &Scheduler{
		workerPool:     workerPool,
		jobRepo:        jobRepo,
		businessRepo:   businessRepo,
		settingsRepo:   settingsRepo,
		keywordRepo:    keywordRepo,
		cfg:            cfg,
		crawler:        enrichment.NewCrawler(nil, userAgent),
		interval:       24 * time.Hour,
		nextRunAt:      nextMidnight,
		autoEnabled:    false,
		maxPagesPerRun: 3,
		pageSize:       20,
		maxResults:     60,
		enrichContacts: true,
		engine:         "places",
		stopChan:       make(chan struct{}),
	}

	if settingsRepo != nil {
		var saved ScheduleInfo
		if err := settingsRepo.Get(context.Background(), "bali_schedule", &saved); err == nil {
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
	s.mu.Lock()
	if !s.autoEnabled {
		s.mu.Unlock()
		return
	}

	if now.After(s.nextRunAt) || now.Equal(s.nextRunAt) {
		slog.Info("Triggering scheduled auto scraper cycle for keywords", "time", now.Format(time.RFC3339))
		s.lastRunAt = &now
		s.nextRunAt = now.Add(s.interval)

		maxPages := s.maxPagesPerRun
		pageSize := s.pageSize
		enrich := s.enrichContacts
		eng := s.engine
		s.mu.Unlock()

		go s.runKeywordScrapes(ctx, nil, maxPages, pageSize, enrich, eng, now)
		return
	}
	s.mu.Unlock()
}

func (s *Scheduler) runKeywordScrapes(
	ctx context.Context,
	queries []string,
	maxPages int,
	pageSize int,
	enrich bool,
	eng string,
	now time.Time,
) []*model.ScrapeJob {
	if s.keywordRepo == nil {
		slog.Warn("Keyword repository is nil, cannot run keyword scrape")
		return nil
	}

	// 1. Resolve API key
	apiKey := ""
	if s.cfg != nil {
		apiKey = s.cfg.GooglePlacesAPIKey
	}
	if s.settingsRepo != nil {
		var savedKey string
		if err := s.settingsRepo.Get(ctx, "google_places_api_key", &savedKey); err == nil && savedKey != "" {
			apiKey = savedKey
		}
	}

	if apiKey == "" {
		slog.Error("Cannot run scheduled scrape: Google Places API key is not configured")
		return nil
	}

	if maxPages <= 0 {
		maxPages = 3
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	// 2. Resolve target keywords
	var targetKeywords []*model.ScraperKeyword

	cleanQueries := []string{}
	for _, q := range queries {
		t := strings.TrimSpace(q)
		if t != "" {
			cleanQueries = append(cleanQueries, t)
		}
	}

	if len(cleanQueries) > 0 {
		for _, q := range cleanQueries {
			// Find or create keyword exactly
			kw, _ := s.keywordRepo.GetByKeyword(ctx, q)
			if kw == nil {
				kw, _ = s.keywordRepo.Create(ctx, q)
			}
			if kw != nil {
				targetKeywords = append(targetKeywords, kw)
			}
		}
	} else {
		// Load active / all keywords
		kws, err := s.keywordRepo.GetActiveKeywords(ctx)
		if err == nil && len(kws) > 0 {
			targetKeywords = kws
		} else {
			allKws, _, _ := s.keywordRepo.List(ctx, model.ListKeywordFilter{
				Page:     1,
				PageSize: 100,
			})
			targetKeywords = allKws
		}
	}

	if len(targetKeywords) == 0 {
		slog.Info("No search keywords configured for auto scraper")
		return nil
	}

	placesClient := gmaps.NewPlacesAPIClient(apiKey)
	var createdJobs []*model.ScrapeJob

	for _, kw := range targetKeywords {
		curToken := kw.NextPageToken
		curPage := kw.CurrentPage
		if curToken == "" || kw.Status == model.KeywordStatusCompleted {
			curToken = ""
			curPage = 1
		}

		// Create job record
		job := &model.ScrapeJob{
			Status:         model.JobStatusProcessing,
			Query:          kw.Keyword,
			Location:       "Google Maps",
			Category:       "Local Business",
			MaxResults:     maxPages * pageSize,
			EnrichContacts: enrich,
			Metadata: map[string]interface{}{
				"source":       "auto_scraper_keyword",
				"keyword_id":   kw.ID.String(),
				"current_page": curPage,
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		if s.jobRepo != nil {
			_ = s.jobRepo.Create(ctx, job)
		}
		createdJobs = append(createdJobs, job)

		// Fetch places with keyword cursor
		leads, nextToken, pagesFetched, cycleFinished, err := placesClient.SearchPlacesTextWithCursor(
			ctx,
			kw.Keyword,
			curToken,
			maxPages,
			pageSize,
		)

		if err != nil {
			if gmaps.IsInvalidPageTokenError(err) && curToken != "" {
				slog.Warn("Page token expired or invalid in scheduler, resetting cursor to page 1 and retrying", "keyword", kw.Keyword, "error", err)
				_ = s.keywordRepo.ResetCursor(ctx, kw.ID)
				curToken = ""
				curPage = 1
				leads, nextToken, pagesFetched, cycleFinished, err = placesClient.SearchPlacesTextWithCursor(
					ctx,
					kw.Keyword,
					"",
					maxPages,
					pageSize,
				)
			}
		}

		if err != nil {
			slog.Error("Failed keyword scrape via scheduler", "keyword", kw.Keyword, "error", err)
			if s.jobRepo != nil {
				_ = s.jobRepo.UpdateStatus(ctx, job.ID, model.JobStatusFailed, 0, 0, 1, fmt.Sprintf("Places error: %v", err))
			}
			continue
		}

		leadsFound := len(leads)

		// Upsert leads into DB
		if s.businessRepo != nil {
			for _, lead := range leads {
				lead.JobID = &job.ID
				if strings.TrimSpace(lead.Category) == "" {
					lead.Category = "Local Business"
				}
				lead.Subcategories = append(lead.Subcategories, kw.Keyword)
				if _, err := s.businessRepo.Upsert(ctx, lead); err != nil {
					slog.Error("Failed to upsert lead from scheduled keyword scrape", "keyword", kw.Keyword, "name", lead.Name, "error", err)
				}
			}
		}

		// Update cursor state in DB
		newCurrentPage := curPage + pagesFetched
		newStatus := model.KeywordStatusPaginating
		if cycleFinished || nextToken == "" {
			newStatus = model.KeywordStatusCompleted
			nextToken = ""
		}

		_ = s.keywordRepo.UpdateCursor(ctx, kw.ID, nextToken, newCurrentPage, leadsFound, newStatus)

		// Mark job completed
		if s.jobRepo != nil {
			_ = s.jobRepo.UpdateStatus(ctx, job.ID, model.JobStatusCompleted, leadsFound, 0, 0, "")
		}

		// Background enrichment
		if enrich && len(leads) > 0 {
			go s.enrichLeadsAsync(leads)
		}
	}

	return createdJobs
}

func (s *Scheduler) enrichLeadsAsync(leads []*model.BusinessLead) {
	if len(leads) == 0 || s.crawler == nil || s.businessRepo == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5)

	for _, lead := range leads {
		if strings.TrimSpace(lead.Website) == "" || lead.EnrichmentStatus == model.EnrichmentSuccess {
			continue
		}

		wg.Add(1)
		semaphore <- struct{}{}

		go func(l *model.BusinessLead) {
			defer wg.Done()
			defer func() { <-semaphore }()
			defer func() {
				if r := recover(); r != nil {
					slog.Error("panic recovered in scheduled enrichment goroutine", "lead", l.Name, "panic", r)
				}
			}()

			if s.crawler == nil {
				return
			}

			res, err := s.crawler.EnrichWebsite(ctx, l.Website)
			if err != nil || res == nil {
				_ = s.businessRepo.UpdateEnrichment(ctx, l.ID, nil, nil, "", "", "", "", "", "", model.EnrichmentFailed)
				return
			}

			_ = s.businessRepo.UpdateEnrichment(
				ctx, l.ID,
				res.Emails, res.Phones,
				res.Socials.Facebook, res.Socials.Instagram, res.Socials.LinkedIn,
				res.Socials.Twitter, res.Socials.YouTube, res.Socials.TikTok,
				model.EnrichmentSuccess,
			)
		}(lead)
	}

	wg.Wait()
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
	}
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

	maxPages := s.maxPagesPerRun
	pageSize := s.pageSize
	eng := s.engine
	s.mu.Unlock()

	if engine != "" {
		eng = engine
	}
	if maxResults > 0 {
		maxPages = (maxResults + 19) / 20
	}

	jobs := s.runKeywordScrapes(ctx, queries, maxPages, pageSize, enrich, eng, now)
	if len(jobs) > 0 {
		return jobs[0], nil
	}

	return nil, nil
}
