package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"business-scraping-service/backend/internal/api"
	"business-scraping-service/backend/internal/api/handler"
	"business-scraping-service/backend/internal/config"
	"business-scraping-service/backend/internal/engine"
	"business-scraping-service/backend/internal/engine/gmaps"
	"business-scraping-service/backend/internal/logger"
	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"
	"business-scraping-service/backend/internal/service"
	"business-scraping-service/backend/internal/session"
	"business-scraping-service/backend/internal/utils"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func main() {
	// 1. Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize structured logging
	logger.InitLogger(cfg.LogLevel, cfg.LogFormat)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 3. Connect to PostgreSQL
	dbPool, err := repository.InitDB(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Printf("Failed to connect to PostgreSQL: %v\n", err)
		slog.Error("Failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// 4. Connect to Redis
	redisOpt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		fmt.Printf("Failed to parse Redis URL: %v\n", err)
		slog.Error("Failed to parse Redis URL", "error", err)
		os.Exit(1)
	}
	redisCli := redis.NewClient(redisOpt)
	if err := redisCli.Ping(ctx).Err(); err != nil {
		fmt.Printf("Failed to connect to Redis: %v\n", err)
		slog.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisCli.Close()

	fmt.Printf("[SERVER] Business Scraping Service Backend Ready\n")
	fmt.Printf("  Port:        %d\n", cfg.Port)
	fmt.Printf("  Environment: %s\n", cfg.NodeEnv)
	fmt.Printf("  Database:    Connected\n")
	fmt.Printf("  Redis:       Connected\n")
	fmt.Printf("[SERVER] Listening on: http://0.0.0.0:%d\n\n", cfg.Port)

	// 5. Initialize Repositories
	adminRepo := repository.NewAdminRepository(dbPool)
	jobRepo := repository.NewJobRepository(dbPool)
	businessRepo := repository.NewBusinessRepository(dbPool)
	settingsRepo := repository.NewSettingsRepository(dbPool)
	keywordRepo := repository.NewKeywordRepository(dbPool)
	areaRepo := repository.NewAreaRepository(dbPool)
	usageRepo := repository.NewUsageRepository(dbPool)

	// 6. Initialize Redis Session Store
	sessionStore := session.NewRedisStore(redisCli, session.DefaultSessionTTL)

	// 7. Auto-bootstrap Super Admin if no admin exists
	bootstrapSuperAdmin(ctx, adminRepo, cfg)

	// 7.1 A key saved on the Settings page takes precedence over the env key.
	var savedAPIKey string
	if err := settingsRepo.Get(ctx, service.GoogleAPIKeySettingKey, &savedAPIKey); err != nil {
		slog.Warn("Failed to load saved Google Places API key, using env key", "error", err)
	} else if savedAPIKey != "" {
		cfg.GooglePlacesAPIKey = savedAPIKey
	}

	if cfg.GooglePlacesBaseURL != "" {
		slog.Warn("Using a non-default Google Places API host", "url", cfg.GooglePlacesBaseURL)
		gmaps.DefaultBaseURL = cfg.GooglePlacesBaseURL
	}

	// 8. Initialize Extraction & Enrichment Engine & Scheduler
	workerPool := engine.NewWorkerPool(cfg, jobRepo, businessRepo, keywordRepo, areaRepo, usageRepo, redisCli)
	var placesBudget int
	if err := settingsRepo.Get(ctx, service.PlacesBudgetSettingKey, &placesBudget); err != nil {
		slog.Warn("Failed to load Places daily budget", "error", err)
	}
	workerPool.SetPlacesBudget(placesBudget)

	schedulerStateRepo := repository.NewSchedulerStateRepository(dbPool)
	var quotaPausedUntil *time.Time
	if st, err := schedulerStateRepo.Get(ctx); err != nil {
		slog.Error("Failed to load scheduler state", "error", err)
	} else {
		quotaPausedUntil = st.QuotaPausedUntil
	}
	quota := engine.NewQuotaGuard(schedulerStateRepo, quotaPausedUntil)

	jobQueue := engine.NewJobQueue(jobRepo, keywordRepo, businessRepo, workerPool, quota, engine.QueueOptions{
		Workers: cfg.MaxConcurrentScrapers,
	})
	jobQueue.Start()

	scheduler := engine.NewScheduler(workerPool, jobQueue, schedulerStateRepo, settingsRepo, keywordRepo, cfg)
	scheduler.Start(ctx)

	// 9. Initialize Services
	authSvc := service.NewAuthService(adminRepo, sessionStore)
	jobSvc := service.NewJobService(jobRepo, jobQueue)
	businessSvc := service.NewBusinessService(businessRepo)
	exportSvc := service.NewExportService(businessRepo)
	metricsSvc := service.NewMetricsService(businessRepo, scheduler)
	systemSvc := service.NewSystemService(cfg, dbPool, redisCli, workerPool, scheduler, settingsRepo, quota)
	keywordSvc := service.NewKeywordService(keywordRepo, areaRepo, jobRepo, jobQueue, workerPool)

	// 10. Initialize Handlers
	handlers := &api.Handlers{
		Auth:     handler.NewAuthHandler(authSvc),
		Job:      handler.NewJobHandler(jobSvc),
		Business: handler.NewBusinessHandler(businessSvc),
		Export:   handler.NewExportHandler(exportSvc),
		Metrics:  handler.NewMetricsHandler(metricsSvc),
		System:   handler.NewSystemHandler(systemSvc),
		Keyword:  handler.NewKeywordHandler(keywordSvc),
		Area:     handler.NewAreaHandler(keywordSvc),
	}

	services := &api.Services{
		Auth: authSvc,
	}

	// 11. Build HTTP Router & Server
	router := api.CreateRouter(cfg, handlers, services)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 12. Graceful Shutdown: stop taking requests, let running jobs finish (or requeue them), then exit.
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
		sig := <-sigChan

		slog.Info("Shutdown signal received, shutting down gracefully...", "signal", sig.String())
		server.SetKeepAlivesEnabled(false)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("Error during HTTP server shutdown", "error", err)
			_ = server.Close()
		}
		jobQueue.Shutdown(cfg.ShutdownGrace)
		cancel()
	}()

	slog.Info(fmt.Sprintf("API Server listening on http://0.0.0.0:%d", cfg.Port))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}

	<-shutdownDone
	slog.Info("Server stopped cleanly")
}

func bootstrapSuperAdmin(ctx context.Context, repo *repository.AdminRepository, cfg *config.Config) {
	count, err := repo.CountAll(ctx)
	if err != nil {
		slog.Warn("Failed to check admin users count", "error", err)
		return
	}

	if count == 0 {
		slog.Info("No admin users found. Bootstrapping initial Super Admin...", "username", cfg.SuperAdminUsername, "email", cfg.SuperAdminEmail)
		hash, err := utils.HashPassword(cfg.SuperAdminPassword)
		if err != nil {
			slog.Error("Failed to hash superadmin password", "error", err)
			return
		}

		admin := &model.AdminUser{
			ID:           uuid.New(),
			Email:        cfg.SuperAdminEmail,
			Username:     cfg.SuperAdminUsername,
			PasswordHash: hash,
			IsActive:     true,
		}

		if err := repo.Create(ctx, admin); err != nil {
			slog.Error("Failed to bootstrap superadmin", "error", err)
			return
		}

		slog.Info("Initial Super Admin bootstrapped successfully!",
			"username", cfg.SuperAdminUsername,
			"email", cfg.SuperAdminEmail,
		)
	}
}
