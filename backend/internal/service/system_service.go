package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"business-scraping-service/backend/internal/config"
	"business-scraping-service/backend/internal/engine"
	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"
)

// GoogleAPIKeySettingKey is where the key set on the Settings page is saved, so it survives restarts.
const GoogleAPIKeySettingKey = "google_places_api_key"

type SystemService struct {
	cfg         *config.Config
	dbPool      *pgxpool.Pool
	redisClient *redis.Client
	workerPool  *engine.WorkerPool
	scheduler   *engine.Scheduler
	settings    *repository.SettingsRepository
}

func NewSystemService(cfg *config.Config, dbPool *pgxpool.Pool, redisClient *redis.Client, workerPool *engine.WorkerPool, scheduler *engine.Scheduler, settings *repository.SettingsRepository) *SystemService {
	return &SystemService{
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
