package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"business-scraping-service/backend/internal/config"
	"business-scraping-service/backend/internal/engine/enrichment"
	"business-scraping-service/backend/internal/engine/gmaps"
	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"
)

type KeywordScrapeResult struct {
	JobID             string              `json:"job_id,omitempty"`
	Keyword           string              `json:"keyword"`
	LeadsFound        int                 `json:"leads_found"`
	PagesTraversed    int                 `json:"pages_traversed"`
	NextPageToken     string              `json:"next_page_token,omitempty"`
	Status            model.KeywordStatus `json:"status"`
	CycleCompleted    bool                `json:"cycle_completed"`
	CurrentPage       int                 `json:"current_page"`
	TotalLeadsScraped int                 `json:"total_leads_scraped"`
	Message           string              `json:"message,omitempty"`
}

type TriggerScrapeOptions struct {
	MaxPages       int  `json:"max_pages"`
	EnrichContacts bool `json:"enrich_contacts"`
}

type KeywordService struct {
	keywordRepo  *repository.KeywordRepository
	businessRepo *repository.BusinessRepository
	jobRepo      *repository.JobRepository
	settingsRepo *repository.SettingsRepository
	crawler      *enrichment.Crawler
	cfg          *config.Config
}

func NewKeywordService(
	keywordRepo *repository.KeywordRepository,
	businessRepo *repository.BusinessRepository,
	jobRepo *repository.JobRepository,
	settingsRepo *repository.SettingsRepository,
	cfg *config.Config,
) *KeywordService {
	return &KeywordService{
		keywordRepo:  keywordRepo,
		businessRepo: businessRepo,
		jobRepo:      jobRepo,
		settingsRepo: settingsRepo,
		crawler:      enrichment.NewCrawler(nil, cfg.DefaultUserAgent),
		cfg:          cfg,
	}
}

func (s *KeywordService) List(ctx context.Context, filter model.ListKeywordFilter) (*model.ListKeywordResponse, error) {
	items, totalCount, err := s.keywordRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	totalPages := 1
	if filter.PageSize > 0 {
		totalPages = int((totalCount + int64(filter.PageSize) - 1) / int64(filter.PageSize))
	}
	if totalPages == 0 {
		totalPages = 1
	}

	return &model.ListKeywordResponse{
		Items:      items,
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	}, nil
}

func (s *KeywordService) Create(ctx context.Context, keyword string) (*model.ScraperKeyword, error) {
	return s.keywordRepo.Create(ctx, keyword)
}

func (s *KeywordService) CreateBulk(ctx context.Context, keywords []string) (int, error) {
	return s.keywordRepo.CreateBulk(ctx, keywords)
}

func (s *KeywordService) ResetCursor(ctx context.Context, id uuid.UUID) error {
	return s.keywordRepo.ResetCursor(ctx, id)
}

func (s *KeywordService) Update(ctx context.Context, id uuid.UUID, keyword string, isActive bool) (*model.ScraperKeyword, error) {
	return s.keywordRepo.Update(ctx, id, keyword, isActive)
}

func (s *KeywordService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.keywordRepo.Delete(ctx, id)
}

func (s *KeywordService) TriggerKeywordScrape(ctx context.Context, id uuid.UUID, opts TriggerScrapeOptions) (*KeywordScrapeResult, error) {
	kw, err := s.keywordRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("keyword not found: %w", err)
	}

	// 1. Resolve API Key & Pagination Settings
	apiKey := s.cfg.GooglePlacesAPIKey
	if s.settingsRepo != nil {
		var savedKey string
		if err := s.settingsRepo.Get(ctx, "google_places_api_key", &savedKey); err == nil && savedKey != "" {
			apiKey = savedKey
		}
	}

	if apiKey == "" {
		return nil, fmt.Errorf("google places api key is not configured")
	}

	maxPages := opts.MaxPages
	if maxPages <= 0 {
		maxPages = 1
	}
	pageSize := 20

	startToken := kw.NextPageToken
	startPage := kw.CurrentPage
	if startToken == "" || kw.Status == model.KeywordStatusCompleted {
		startToken = ""
		startPage = 1
	}

	// 2. Create Scrape Job for tracking and live history
	jobID := uuid.New()
	startTime := time.Now().UTC()
	job := &model.ScrapeJob{
		ID:             jobID,
		Status:         model.JobStatusProcessing,
		Query:          kw.Keyword,
		Location:       "Google Maps",
		Category:       "Keyword Scrape",
		MaxResults:     maxPages * pageSize,
		EnrichContacts: opts.EnrichContacts,
		StartedAt:      &startTime,
	}
	if s.jobRepo != nil {
		_ = s.jobRepo.Create(ctx, job)
	}

	// 3. Mark keyword status as processing immediately in DB
	_ = s.keywordRepo.SetStatus(ctx, kw.ID, model.KeywordStatusProcessing)

	// 4. Run scraping in background goroutine to avoid freezing UI
	go func(targetKw *model.ScraperKeyword, targetJobID uuid.UUID, key string, pages int, enrich bool, initialToken string, initialPage int) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic recovered in keyword scrape worker", "keyword", targetKw.Keyword, "panic", r)
				_ = s.keywordRepo.SetStatus(context.Background(), targetKw.ID, model.KeywordStatusReady)
			}
		}()

		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		curToken := initialToken
		curPage := initialPage

		placesClient := gmaps.NewPlacesAPIClient(key)
		leads, nextToken, pagesFetched, cycleFinished, fetchErr := placesClient.SearchPlacesTextWithCursor(bgCtx, targetKw.Keyword, curToken, pages, pageSize)
		if fetchErr != nil {
			// Check if the failure was due to an expired or invalid pageToken
			if gmaps.IsInvalidPageTokenError(fetchErr) && curToken != "" {
				slog.Warn("Page token expired or invalid, resetting cursor to page 1 and retrying", "keyword", targetKw.Keyword, "error", fetchErr)
				_ = s.keywordRepo.ResetCursor(bgCtx, targetKw.ID)
				curToken = ""
				curPage = 1
				leads, nextToken, pagesFetched, cycleFinished, fetchErr = placesClient.SearchPlacesTextWithCursor(bgCtx, targetKw.Keyword, "", pages, pageSize)
			}
		}

		if fetchErr != nil {
			slog.Error("Google Places fetch failed for keyword", "keyword", targetKw.Keyword, "error", fetchErr)
			if s.jobRepo != nil {
				_ = s.jobRepo.UpdateStatus(bgCtx, targetJobID, model.JobStatusFailed, 0, 0, 1, fetchErr.Error())
			}
			_ = s.keywordRepo.SetStatus(bgCtx, targetKw.ID, model.KeywordStatusReady)
			return
		}

		// Ingest leads into DB
		newCount := 0
		for _, lead := range leads {
			if strings.TrimSpace(lead.Category) == "" {
				lead.Category = "Local Business"
			}
			lead.JobID = &targetJobID
			lead.ScrapeKeywords = append(lead.ScrapeKeywords, targetKw.Keyword)
			wasNew, upsertErr := s.businessRepo.Upsert(bgCtx, lead)
			if upsertErr != nil {
				slog.Error("Failed to upsert lead from keyword scrape", "keyword", targetKw.Keyword, "name", lead.Name, "error", upsertErr)
			}
			if wasNew {
				newCount++
			}
		}

		// Update Job status to Completed (number of leads retrieved in this scrape run)
		if s.jobRepo != nil {
			_ = s.jobRepo.UpdateStatus(bgCtx, targetJobID, model.JobStatusCompleted, len(leads), 0, 0, "")
		}

		// Update Cursor state in DB
		newCurrentPage := curPage + pagesFetched
		newStatus := model.KeywordStatusPaginating
		if cycleFinished || nextToken == "" {
			newStatus = model.KeywordStatusCompleted
			nextToken = ""
		}

		if err := s.keywordRepo.UpdateCursor(bgCtx, targetKw.ID, nextToken, newCurrentPage, len(leads), newStatus); err != nil {
			slog.Error("Failed to update keyword cursor state", "keyword_id", targetKw.ID, "error", err)
		}

		// Background Enrichment for leads with websites
		if enrich && len(leads) > 0 {
			s.enrichLeadsAsync(leads, targetJobID)
		}
	}(kw, jobID, apiKey, maxPages, opts.EnrichContacts, startToken, startPage)

	return &KeywordScrapeResult{
		JobID:             jobID.String(),
		Keyword:           kw.Keyword,
		Status:            model.KeywordStatusProcessing,
		CurrentPage:       startPage,
		TotalLeadsScraped: kw.TotalLeadsScraped,
		Message:           "Scrape started in background",
	}, nil
}

func (s *KeywordService) enrichLeadsAsync(leads []*model.BusinessLead, jobID uuid.UUID) {
	if len(leads) == 0 {
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
					slog.Error("panic recovered in keyword enrichment goroutine", "lead", l.Name, "panic", r)
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

			if s.jobRepo != nil && len(res.Emails) > 0 {
				_ = s.jobRepo.IncrementProgress(context.Background(), jobID, 0, 1, 0)
			}
		}(lead)
	}

	wg.Wait()
}
