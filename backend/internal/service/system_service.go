package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"business-scraping-service/backend/internal/config"
	"business-scraping-service/backend/internal/engine"
	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// GoogleAPIKeySettingKey is where the key set on the Settings page is saved, so it survives restarts.
const GoogleAPIKeySettingKey = "google_places_api_key"

// PlacesBudgetSettingKey stores the daily Places request budget (0 = no budget).
const PlacesBudgetSettingKey = "places_daily_budget"

type SystemService struct {
	cfg         *config.Config
	dbPool      *pgxpool.Pool
	redisClient *redis.Client
	workerPool  *engine.WorkerPool
	scheduler   *engine.Scheduler
	settings    *repository.SettingsRepository
	quota       *engine.QuotaGuard
	queue       *engine.JobQueue
	jobs        *repository.JobRepository
	keywords    *repository.KeywordRepository
}

func NewSystemService(cfg *config.Config, dbPool *pgxpool.Pool, redisClient *redis.Client, workerPool *engine.WorkerPool, scheduler *engine.Scheduler, settings *repository.SettingsRepository, quota *engine.QuotaGuard, queue *engine.JobQueue, jobs *repository.JobRepository, keywords *repository.KeywordRepository) *SystemService {
	return &SystemService{
		queue:       queue,
		jobs:        jobs,
		keywords:    keywords,
		quota:       quota,
		settings:    settings,
		cfg:         cfg,
		dbPool:      dbPool,
		redisClient: redisClient,
		workerPool:  workerPool,
		scheduler:   scheduler,
	}
}

type SystemSettingsResponse struct {
	HasGoogleAPIKey      bool                 `json:"has_google_api_key"`
	GoogleAPIKeyMasked   string               `json:"google_api_key_masked"`
	DeduplicationMode    string               `json:"deduplication_mode"`
	MaxConcurrentWorkers int                  `json:"max_concurrent_workers"`
	PlacesDailyBudget    int                  `json:"places_daily_budget"`
	PlacesRequestsToday  int                  `json:"places_requests_today"`
	QuotaResetAt         time.Time            `json:"quota_reset_at"`
	Schedule             *engine.ScheduleInfo `json:"schedule,omitempty"`
}

func (s *SystemService) GetSettings() *SystemSettingsResponse {
	hasKey := s.cfg.GooglePlacesAPIKey != ""
	masked := ""
	if hasKey {
		key := s.cfg.GooglePlacesAPIKey
		if len(key) > 8 {
			masked = key[:4] + "..." + key[len(key)-4:]
		} else {
			masked = "****"
		}
	}

	var sched *engine.ScheduleInfo
	if s.scheduler != nil {
		info := s.scheduler.GetScheduleInfo()
		sched = &info
	}

	return &SystemSettingsResponse{
		HasGoogleAPIKey:      hasKey,
		GoogleAPIKeyMasked:   masked,
		DeduplicationMode:    "SMART_MERGE_UPSERT",
		MaxConcurrentWorkers: s.cfg.MaxConcurrentScrapers,
		PlacesDailyBudget:    s.workerPool.PlacesBudget(),
		PlacesRequestsToday:  s.placesRequestsToday(),
		QuotaResetAt:         engine.NextQuotaReset(time.Now()),
		Schedule:             sched,
	}
}

// UpdateGoogleAPIKey saves the key and switches every scrape path to it. Before, it only changed the
// in-memory config, so a restart silently fell back to the env key.
func (s *SystemService) UpdateGoogleAPIKey(ctx context.Context, apiKey string) error {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return fmt.Errorf("api key must not be empty")
	}
	if s.settings != nil {
		if err := s.settings.Set(ctx, GoogleAPIKeySettingKey, apiKey); err != nil {
			return fmt.Errorf("failed to save api key: %w", err)
		}
	}
	s.cfg.GooglePlacesAPIKey = apiKey
	s.workerPool.SetGoogleAPIKey(apiKey)
	// A new key may belong to a project with quota left: let scraping try again.
	if s.quota != nil {
		s.quota.Clear(ctx)
	}
	return nil
}

func (s *SystemService) GetSchedule() *engine.ScheduleInfo {
	if s.scheduler == nil {
		return nil
	}
	info := s.scheduler.GetScheduleInfo()
	return &info
}

func (s *SystemService) UpdateSchedule(
	intervalHours int,
	autoEnabled bool,
	queries []string,
	locations []string,
	categories []string,
	onlyNewBusinesses bool,
	requireWebsite bool,
	websiteFilter string,
	recencyFilter string,
	maxReviewCountForNew int,
	maxPagesPerRun int,
	pageSize int,
	maxResults int,
	enrich bool,
	engine string,
) *engine.ScheduleInfo {
	if s.scheduler == nil {
		return nil
	}
	info := s.scheduler.UpdateSchedule(intervalHours, autoEnabled, queries, locations, categories, onlyNewBusinesses, requireWebsite, websiteFilter, recencyFilter, maxReviewCountForNew, maxPagesPerRun, pageSize, maxResults, enrich, engine)
	return &info
}

func (s *SystemService) TriggerImmediateScrape(
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
	if s.scheduler == nil {
		return nil, nil
	}
	return s.scheduler.TriggerNow(ctx, queries, locations, categories, maxResults, enrich, engine, onlyNew, requireWebsite, websiteFilter, recencyFilter, maxReviews)
}

type HealthResponse struct {
	Status      string `json:"status"`
	Environment string `json:"environment"`
	Database    string `json:"database"`
	Redis       string `json:"redis"`
	Timestamp   string `json:"timestamp"`
}

func (s *SystemService) GetHealth(ctx context.Context) *HealthResponse {
	dbStatus := "connected"
	if err := s.dbPool.Ping(ctx); err != nil {
		dbStatus = "disconnected: " + err.Error()
	}

	redisStatus := "connected"
	if err := s.redisClient.Ping(ctx).Err(); err != nil {
		redisStatus = "disconnected: " + err.Error()
	}

	overall := "healthy"
	if dbStatus != "connected" || redisStatus != "connected" {
		overall = "degraded"
	}

	return &HealthResponse{
		Status:      overall,
		Environment: s.cfg.NodeEnv,
		Database:    dbStatus,
		Redis:       redisStatus,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
}

func (s *SystemService) placesRequestsToday() int {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	n, err := s.workerPool.PlacesRequestsToday(ctx)
	if err != nil {
		return 0
	}
	return n
}

// SetPlacesBudget saves the daily Places request budget (0 = no budget). Raising it lifts a pause
// caused by the old budget.
func (s *SystemService) SetPlacesBudget(ctx context.Context, n int) error {
	if n < 0 {
		return fmt.Errorf("budget cannot be negative")
	}
	if s.settings != nil {
		if err := s.settings.Set(ctx, PlacesBudgetSettingKey, n); err != nil {
			return fmt.Errorf("failed to save budget: %w", err)
		}
	}
	s.workerPool.SetPlacesBudget(n)
	if s.quota != nil {
		s.quota.Clear(ctx)
	}
	return nil
}

// ScrapeHealth is the health panel: queue, recent outcomes, quota/budget, scheduler and keywords.
type ScrapeHealth struct {
	Status              string                    `json:"status"` // ok | warning | paused
	Issues              []string                  `json:"issues"`
	Jobs                *repository.JobHealth     `json:"jobs"`
	Keywords            *repository.KeywordHealth `json:"keywords"`
	QuotaPausedUntil    *time.Time                `json:"quota_paused_until,omitempty"`
	QuotaResetAt        time.Time                 `json:"quota_reset_at"`
	PlacesRequestsToday int                       `json:"places_requests_today"`
	PlacesDailyBudget   int                       `json:"places_daily_budget"`
	AutoEnabled         bool                      `json:"auto_enabled"`
	NextRunAt           time.Time                 `json:"next_run_at"`
	LastRunAt           *time.Time                `json:"last_run_at,omitempty"`
	HasAPIKey           bool                      `json:"has_api_key"`
	AlertsEnabled       bool                      `json:"alerts_enabled"`
}

func (s *SystemService) ScrapeHealth(ctx context.Context) (*ScrapeHealth, error) {
	jobs, err := s.jobs.Health(ctx)
	if err != nil {
		return nil, err
	}
	kws, err := s.keywords.Health(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	h := &ScrapeHealth{
		Status:              "ok",
		Issues:              []string{},
		Jobs:                jobs,
		Keywords:            kws,
		QuotaResetAt:        engine.NextQuotaReset(now),
		PlacesRequestsToday: s.placesRequestsToday(),
		PlacesDailyBudget:   s.workerPool.PlacesBudget(),
		HasAPIKey:           s.workerPool.APIKey() != "",
		AlertsEnabled:       s.queue != nil && s.queue.Alerts().Enabled(),
	}
	if s.scheduler != nil {
		info := s.scheduler.GetScheduleInfo()
		h.AutoEnabled, h.NextRunAt, h.LastRunAt = info.AutoEnabled, info.NextRunAt, info.LastRunAt
	}
	if until, paused := s.quota.PausedUntil(now); paused {
		h.QuotaPausedUntil = &until
		h.Status = "paused"
		h.Issues = append(h.Issues, "Google Places quota or daily budget reached: scraping paused until "+until.Format("2006-01-02 15:04 MST"))
	}
	warn := func(msg string) {
		if h.Status == "ok" {
			h.Status = "warning"
		}
		h.Issues = append(h.Issues, msg)
	}
	if !h.HasAPIKey {
		warn("No Google Places API key configured")
	}
	if jobs.Failed24h > 0 {
		warn(fmt.Sprintf("%d job(s) failed in the last 24 hours", jobs.Failed24h))
	}
	if kws.NoResults > 0 {
		warn(fmt.Sprintf("%d keyword(s) return no results: check their wording or type", kws.NoResults))
	}
	if h.PlacesDailyBudget > 0 && h.PlacesRequestsToday*10 >= h.PlacesDailyBudget*9 && h.QuotaPausedUntil == nil {
		warn(fmt.Sprintf("Daily request budget almost used: %d of %d", h.PlacesRequestsToday, h.PlacesDailyBudget))
	}
	if h.AutoEnabled && kws.Active == 0 {
		warn("Auto scraping is on but no keyword is active, so every keyword is run")
	}
	return h, nil
}
