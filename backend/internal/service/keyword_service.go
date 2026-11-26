package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"business-scraping-service/backend/internal/engine"
	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"

	"github.com/google/uuid"
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
	keywordRepo *repository.KeywordRepository
	jobRepo     *repository.JobRepository
	runner      *engine.KeywordRunner
}

func NewKeywordService(
	keywordRepo *repository.KeywordRepository,
	jobRepo *repository.JobRepository,
	runner *engine.KeywordRunner,
) *KeywordService {
	return &KeywordService{
		keywordRepo: keywordRepo,
		jobRepo:     jobRepo,
		runner:      runner,
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
	if kw.Status == model.KeywordStatusProcessing {
		return nil, fmt.Errorf("keyword %q is already being scraped", kw.Keyword)
	}
	if s.runner.APIKey() == "" {
		return nil, engine.ErrNoAPIKey
	}

	maxPages := opts.MaxPages
	if maxPages <= 0 {
		maxPages = 1
	}
	const pageSize = 20
	_, startPage := engine.StartPosition(kw)

	startTime := time.Now().UTC()
	job := &model.ScrapeJob{
		ID:             uuid.New(),
		Status:         model.JobStatusProcessing,
		Query:          kw.Keyword,
		Location:       "Google Maps",
		Category:       "Keyword Scrape",
		MaxResults:     maxPages * pageSize,
		EnrichContacts: opts.EnrichContacts,
		StartedAt:      &startTime,
	}
	if err := s.jobRepo.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create scrape job: %w", err)
	}
	if err := s.keywordRepo.SetStatus(ctx, kw.ID, model.KeywordStatusProcessing); err != nil {
		return nil, fmt.Errorf("failed to mark keyword processing: %w", err)
	}

	// Run in the background so the UI is not blocked; the runner always leaves the job and keyword in a final state.
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic recovered in keyword scrape worker", "keyword", kw.Keyword, "panic", r)
				_ = s.keywordRepo.SetStatus(context.Background(), kw.ID, model.KeywordStatusReady)
			}
		}()
		_, _ = s.runner.Run(bgCtx, kw, job.ID, engine.KeywordRunOptions{
			MaxPages: maxPages,
			PageSize: pageSize,
			Enrich:   opts.EnrichContacts,
		})
	}()

	return &KeywordScrapeResult{
		JobID:             job.ID.String(),
		Keyword:           kw.Keyword,
		Status:            model.KeywordStatusProcessing,
		CurrentPage:       startPage,
		TotalLeadsScraped: kw.TotalLeadsScraped,
		Message:           "Scrape started in background",
	}, nil
}
