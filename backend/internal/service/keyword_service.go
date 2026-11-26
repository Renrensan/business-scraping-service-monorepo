package service

import (
	"context"
	"errors"
	"fmt"
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
	queue       *engine.JobQueue
	runner      *engine.KeywordRunner
}

func NewKeywordService(
	keywordRepo *repository.KeywordRepository,
	jobRepo *repository.JobRepository,
	queue *engine.JobQueue,
	runner *engine.KeywordRunner,
) *KeywordService {
	return &KeywordService{
		keywordRepo: keywordRepo,
		jobRepo:     jobRepo,
		queue:       queue,
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

// ErrKeywordBusy is returned when the keyword already has a queued or running scrape.
var ErrKeywordBusy = engine.ErrKeywordBusy

func (s *KeywordService) TriggerKeywordScrape(ctx context.Context, id uuid.UUID, opts TriggerScrapeOptions) (*KeywordScrapeResult, error) {
	kw, err := s.keywordRepo.GetByID(ctx, id)
	if err != nil || kw == nil {
		return nil, fmt.Errorf("keyword not found: %w", err)
	}
	if s.runner.APIKey() == "" {
		return nil, engine.ErrNoAPIKey
	}
	if until, paused := s.queue.Quota().PausedUntil(time.Now()); paused {
		return nil, fmt.Errorf("google places api daily quota exceeded; scraping is paused until %s", until.Format("2006-01-02 15:04 MST"))
	}

	maxPages := opts.MaxPages
	if maxPages <= 0 {
		maxPages = 1
	}
	_, startPage := engine.StartPosition(kw)

	job, err := s.queue.EnqueueKeyword(ctx, kw, engine.KeywordRunOptions{
		MaxPages: maxPages,
		PageSize: 20,
		Enrich:   opts.EnrichContacts,
	}, "keyword_trigger")
	if err != nil {
		if errors.Is(err, engine.ErrKeywordBusy) {
			return nil, fmt.Errorf("keyword %q is already being scraped: %w", kw.Keyword, err)
		}
		return nil, fmt.Errorf("failed to queue keyword scrape: %w", err)
	}

	return &KeywordScrapeResult{
		JobID:             job.ID.String(),
		Keyword:           kw.Keyword,
		Status:            model.KeywordStatusProcessing,
		CurrentPage:       startPage,
		TotalLeadsScraped: kw.TotalLeadsScraped,
		Message:           "Scrape queued",
	}, nil
}
