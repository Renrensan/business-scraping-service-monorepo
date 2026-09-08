package service

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"business-scraping-service/backend/internal/config"
)

type SystemService struct {
	cfg         *config.Config
	dbPool      *pgxpool.Pool
	redisClient *redis.Client
}

func NewSystemService(cfg *config.Config, dbPool *pgxpool.Pool, redisClient *redis.Client) *SystemService {
	return &SystemService{
		cfg:         cfg,
		dbPool:      dbPool,
		redisClient: redisClient,
	}
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
