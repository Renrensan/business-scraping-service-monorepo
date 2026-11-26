package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"business-scraping-service/backend/internal/api"
	"business-scraping-service/backend/internal/api/handler"
	"business-scraping-service/backend/internal/api/middleware"
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
		// The logger is configured from cfg, so this uses slog's default (stderr).
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// 2. Initialize structured logging
	logger.InitLogger(cfg.LogLevel, cfg.LogFormat)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 3. Connect to PostgreSQL
	dbPool, err := repository.InitDB(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// 4. Connect to Redis
	redisOpt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		slog.Error("Failed to parse Redis URL", "error", err)
		os.Exit(1)
	}
	redisCli := redis.NewClient(redisOpt)
	if err := redisCli.Ping(ctx).Err(); err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisCli.Close()

	slog.Info("Backend ready: database and Redis connected", "port", cfg.Port, "environment", cfg.NodeEnv)

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

	adminSvc := service.NewAdminService(adminRepo, sessionStore)
	auditRepo := repository.NewAuditRepository(dbPool)
	auditRecorder := middleware.NewRecorder(auditRepo)

	// One-off command: create or reset an admin, then exit (no workers are started).
	//   echo -n 'password' | /app/server admin-upsert --email someone@example.com [--username name]
	if len(os.Args) > 1 && os.Args[1] == "admin-upsert" {
		os.Exit(runAdminUpsert(ctx, adminSvc, os.Args[2:]))
	}

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
	alerter := engine.NewAlerter(cfg.EvolutionAPIURL, cfg.EvolutionAPIKey, cfg.EvolutionInstance, cfg.AlertWhatsAppNumber)
	if alerter.Enabled() {
		slog.Info("WhatsApp alerts enabled", "instance", cfg.EvolutionInstance)
	}
	jobQueue.SetAlerter(alerter)

	// Lead scoring (e.g. GuestChatPro): refreshed after scraping/enrichment, and fully at startup.
	scoringRepo := repository.NewScoringRepository(dbPool)
	contactRepo := repository.NewContactRepository(dbPool)
	scorer := engine.NewScorer(scoringRepo, areaRepo)
	workerPool.SetScorer(scorer)
	scorer.RescoreAllInBackground()
	jobQueue.Start()

	scheduler := engine.NewScheduler(workerPool, jobQueue, schedulerStateRepo, settingsRepo, keywordRepo, cfg)
	scheduler.Start(ctx)

	// 9. Initialize Services
	authSvc := service.NewAuthService(adminRepo, sessionStore)
	jobSvc := service.NewJobService(jobRepo, jobQueue)
	businessSvc := service.NewBusinessService(businessRepo, scoringRepo, contactRepo)
	exportSvc := service.NewExportService(businessRepo)
	metricsSvc := service.NewMetricsService(businessRepo, scheduler)
	systemSvc := service.NewSystemService(cfg, dbPool, redisCli, workerPool, scheduler, settingsRepo, quota, jobQueue, jobRepo, keywordRepo)
	keywordSvc := service.NewKeywordService(keywordRepo, areaRepo, jobRepo, jobQueue, workerPool)
	keywordSvc.OnAreasChanged = scorer.RescoreAllInBackground

	// 10. Initialize Handlers
	handlers := &api.Handlers{
		Auth:     handler.NewAuthHandler(authSvc, auditRecorder),
		Job:      handler.NewJobHandler(jobSvc),
		Business: handler.NewBusinessHandler(businessSvc),
		Export:   handler.NewExportHandler(exportSvc),
		Metrics:  handler.NewMetricsHandler(metricsSvc),
		System:   handler.NewSystemHandler(systemSvc),
		Keyword:  handler.NewKeywordHandler(keywordSvc),
		Area:     handler.NewAreaHandler(keywordSvc),
		Scoring:  handler.NewScoringHandler(service.NewScoringService(scoringRepo, scorer)),
		Contact:  handler.NewContactHandler(service.NewContactService(contactRepo)),
		Admin:    handler.NewAdminHandler(adminSvc, auditRepo),
	}

	services := &api.Services{
		Auth:  authSvc,
		Audit: auditRecorder,
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
			if err := server.Close(); err != nil {
				slog.Error("Error closing HTTP server", "error", err)
			}
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

// runAdminUpsert creates an admin or resets an existing one's password (and reactivates it).
// The password is read from stdin so it doesn't appear in the process list or shell history.
func runAdminUpsert(ctx context.Context, admins *service.AdminService, args []string) int {
	fs := flag.NewFlagSet("admin-upsert", flag.ContinueOnError)
	email := fs.String("email", "", "admin email (required)")
	username := fs.String("username", "", "username (default: part of the email before @)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	pw, err := io.ReadAll(io.LimitReader(os.Stdin, 1024))
	if err != nil {
		slog.Error("reading password from stdin", "error", err)
		return 1
	}
	password := strings.TrimRight(string(pw), "\r\n")
	a, created, err := admins.Upsert(ctx, service.CreateAdminInput{Email: *email, Username: *username, Password: password})
	if err != nil {
		slog.Error("admin-upsert failed", "email", *email, "error", err)
		return 1
	}
	action := "updated (password reset, active)"
	if created {
		action = "created"
	}
	slog.Info("admin "+action, "username", a.Username, "email", a.Email, "id", a.ID)
	return 0
}
