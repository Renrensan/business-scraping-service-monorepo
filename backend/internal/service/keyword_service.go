package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"business-scraping-service/backend/internal/apperr"
	"business-scraping-service/backend/internal/engine"
	"business-scraping-service/backend/internal/engine/gmaps"
	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	areaRepo    *repository.AreaRepository
	jobRepo     *repository.JobRepository
	queue       *engine.JobQueue
	wp          *engine.WorkerPool
	runner      *engine.KeywordRunner

	// OnAreasChanged runs after an area is added or deleted (lead areas for scoring may change).
	OnAreasChanged func()
}

func NewKeywordService(
	keywordRepo *repository.KeywordRepository,
	areaRepo *repository.AreaRepository,
	jobRepo *repository.JobRepository,
	queue *engine.JobQueue,
	wp *engine.WorkerPool,
) *KeywordService {
	return &KeywordService{
		keywordRepo: keywordRepo,
		areaRepo:    areaRepo,
		jobRepo:     jobRepo,
		queue:       queue,
		wp:          wp,
		runner:      wp.KeywordRunner(),
	}
}

// AreaKeywordsResult reports what CreateAreaKeywords did.
type AreaKeywordsResult struct {
	Created  int      `json:"created"`
	Existing int      `json:"existing"`
	Keywords []string `json:"keywords"`
}

// CreateAreaKeywords creates one "term · area" keyword per area.
func (s *KeywordService) CreateAreaKeywords(ctx context.Context, term, includedType string, areaIDs []uuid.UUID) (*AreaKeywordsResult, error) {
	term = strings.TrimSpace(term)
	if term == "" {
		return nil, apperr.Invalid("term is required")
	}
	if len(areaIDs) == 0 {
		return nil, apperr.Invalid("pick at least one area")
	}
	if strings.ContainsAny(includedType, " ,;") {
		return nil, apperr.Invalid("type must be a single Places type such as lodging")
	}
	res := &AreaKeywordsResult{}
	for _, id := range areaIDs {
		area, err := s.areaRepo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if area == nil {
			return nil, apperr.NotFound("area %s not found", id)
		}
		kw, created, err := s.keywordRepo.CreateAreaKeyword(ctx, term, strings.TrimSpace(strings.ToLower(includedType)), area)
		if err != nil {
			return nil, err
		}
		if created {
			res.Created++
		} else {
			res.Existing++
		}
		res.Keywords = append(res.Keywords, kw.Keyword)
	}
	return res, nil
}

// SuggestType returns the Places type the form pre-fills for a term.
func (s *KeywordService) SuggestType(term string) string {
	return gmaps.SuggestIncludedType(term)
}

func (s *KeywordService) ListAreas(ctx context.Context) ([]*model.Area, error) {
	return s.areaRepo.List(ctx)
}

// AddAreaRequest adds an area either by looking up a place name in Google (one Places request) or
// from corner coordinates.
type AddAreaRequest struct {
	Name    string   `json:"name"`
	LowLat  *float64 `json:"low_lat,omitempty"`
	LowLng  *float64 `json:"low_lng,omitempty"`
	HighLat *float64 `json:"high_lat,omitempty"`
	HighLng *float64 `json:"high_lng,omitempty"`
}

func (s *KeywordService) AddArea(ctx context.Context, req AddAreaRequest) (*model.Area, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, apperr.Invalid("area name is required")
	}
	if existing, err := s.areaRepo.GetByName(ctx, name); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}

	area := &model.Area{Name: name}
	if req.LowLat != nil && req.LowLng != nil && req.HighLat != nil && req.HighLng != nil {
		area.LowLat, area.LowLng, area.HighLat, area.HighLng = *req.LowLat, *req.LowLng, *req.HighLat, *req.HighLng
		area.Source = "manual"
	} else {
		key := s.runner.APIKey()
		if key == "" {
			return nil, engine.ErrNoAPIKey
		}
		if until, paused := s.queue.Quota().PausedUntil(time.Now()); paused {
			return nil, apperr.Unavailable("google places api daily quota exceeded; area lookup is paused until %s", until.Format("2006-01-02 15:04 MST"))
		}
		rect, _, err := s.wp.PlacesClient(key).LookupArea(ctx, name)
		if err != nil {
			if gmaps.IsQuotaError(err) {
				return nil, apperr.Unavailable("google places api daily quota exceeded; try adding the area after the quota resets")
			}
			return nil, fmt.Errorf("%w: %v", apperr.Invalid("could not find %q on Google Maps", name), err)
		}
		area.LowLat, area.LowLng = rect.Low.Latitude, rect.Low.Longitude
		area.HighLat, area.HighLng = rect.High.Latitude, rect.High.Longitude
		area.Source = "lookup"
	}
	if err := s.areaRepo.Create(ctx, area); err != nil {
		return nil, err
	}
	if s.OnAreasChanged != nil {
		s.OnAreasChanged()
	}
	return area, nil
}

func (s *KeywordService) DeleteArea(ctx context.Context, id uuid.UUID) error {
	if err := s.areaRepo.Delete(ctx, id); err != nil {
		return err
	}
	if s.OnAreasChanged != nil {
		s.OnAreasChanged()
	}
	return nil
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
		if err == nil || errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("keyword not found")
		}
		return nil, fmt.Errorf("loading keyword: %w", err)
	}
	if s.runner.APIKey() == "" {
		return nil, engine.ErrNoAPIKey
	}
	if until, paused := s.queue.Quota().PausedUntil(time.Now()); paused {
		return nil, apperr.Unavailable("google places api daily quota exceeded; scraping is paused until %s", until.Format("2006-01-02 15:04 MST"))
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
