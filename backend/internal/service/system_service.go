package service

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"business-scraping-service/backend/internal/config"
	"business-scraping-service/backend/internal/engine"
)

type SystemService struct {
	cfg         *config.Config
	dbPool      *pgxpool.Pool
	redisClient *redis.Client
	workerPool  *engine.WorkerPool
}

func NewSystemService(cfg *config.Config, dbPool *pgxpool.Pool, redisClient *redis.Client, workerPool *engine.WorkerPool) *SystemService {
	return &SystemService{
		cfg:         cfg,
		dbPool:      dbPool,
		redisClient: redisClient,
		workerPool:  workerPool,
	}
}

type SystemSettingsResponse struct {
	HasGoogleAPIKey     bool   `json:"has_google_api_key"`
	GoogleAPIKeyMasked  string `json:"google_api_key_masked"`
	DeduplicationMode   string `json:"deduplication_mode"`
	MaxConcurrentWorkers int    `json:"max_concurrent_workers"`
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

	return &SystemSettingsResponse{
		HasGoogleAPIKey:      hasKey,
		GoogleAPIKeyMasked:   masked,
		DeduplicationMode:    "SMART_MERGE_UPSERT",
		MaxConcurrentWorkers: s.cfg.MaxConcurrentScrapers,
	}
}

func (s *SystemService) UpdateGoogleAPIKey(apiKey string) {
	s.cfg.GooglePlacesAPIKey = apiKey
	s.workerPool.SetGoogleAPIKey(apiKey)
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
