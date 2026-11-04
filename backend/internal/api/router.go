package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"business-scraping-service/backend/internal/api/handler"
	"business-scraping-service/backend/internal/api/middleware"
	"business-scraping-service/backend/internal/config"
	"business-scraping-service/backend/internal/service"
)

type Handlers struct {
	Auth     *handler.AuthHandler
	Job      *handler.JobHandler
	Business *handler.BusinessHandler
	Export   *handler.ExportHandler
	Metrics  *handler.MetricsHandler
	System   *handler.SystemHandler
	Keyword  *handler.KeywordHandler
}

type Services struct {
	Auth *service.AuthService
}

func CreateRouter(cfg *config.Config, h *Handlers, s *Services) http.Handler {
	r := chi.NewRouter()

	// Global Middlewares
	r.Use(middleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(middleware.Recovery)
	r.Use(middleware.HTTPLogger)

	// CORS Setup
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.FrontendURL, "https://admin.momos.asia", "http://admin.momos.asia", "http://localhost:5173", "http://localhost:3000", "http://127.0.0.1:5173", "http://127.0.0.1:3004"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Root Healthcheck (supports GET and HEAD for docker healthchecks)
	r.Get("/health", h.System.GetHealth)
	r.Head("/health", h.System.GetHealth)

	registerRoutes := func(api chi.Router) {
		// Public Auth
		api.Post("/auth/login", h.Auth.Login)
		api.Get("/system/health", h.System.GetHealth)
		api.Head("/system/health", h.System.GetHealth)

		// Protected Routes (Session Required)
		api.Group(func(protected chi.Router) {
			protected.Use(middleware.AuthMiddleware(s.Auth))

			// Auth Me & Logout
			protected.Get("/auth/me", h.Auth.GetMe)
			protected.Post("/auth/logout", h.Auth.Logout)
			protected.Post("/auth/change-password", h.Auth.ChangePassword)

			// Overview & Analytics
			protected.Get("/metrics/overview", h.Metrics.GetOverview)
			protected.Get("/metrics/categories", h.Metrics.GetCategoryStats)
			protected.Get("/metrics/locations", h.Metrics.GetLocationStats)

			// Scrape Jobs
			protected.Post("/jobs", h.Job.Create)
			protected.Get("/jobs", h.Job.List)
			protected.Get("/jobs/{id}", h.Job.GetByID)

			// Scraper Keywords & Cursor Registry
			protected.Get("/keywords", h.Keyword.List)
			protected.Post("/keywords", h.Keyword.Create)
			protected.Put("/keywords/{id}", h.Keyword.Update)
			protected.Patch("/keywords/{id}", h.Keyword.Update)
			protected.Post("/keywords/{id}/scrape", h.Keyword.Trigger)
			protected.Post("/keywords/{id}/trigger", h.Keyword.Trigger)
			protected.Post("/keywords/{id}/reset", h.Keyword.Reset)
			protected.Delete("/keywords/{id}", h.Keyword.Delete)

			// Businesses / Leads Database
			protected.Get("/businesses", h.Business.List)
			protected.Get("/businesses/keywords", h.Business.GetKeywords)
			protected.Get("/businesses/categories", h.Business.GetCategories)
			protected.Get("/businesses/{id}", h.Business.GetByID)
			protected.Patch("/businesses/{id}/contacted", h.Business.PatchContacted)
			protected.Delete("/businesses/{id}", h.Business.Delete)

			// Export
			protected.Get("/export/csv", h.Export.ExportCSV)

			// System Settings & Scheduler
			protected.Get("/system/settings", h.System.GetSettings)
			protected.Post("/system/api-key", h.System.UpdateAPIKey)
			protected.Get("/system/schedule", h.System.GetSchedule)
			protected.Post("/system/schedule", h.System.UpdateSchedule)
			protected.Post("/system/trigger-scrape", h.System.TriggerImmediateScrape)
		})
	}

	// Mount on all common prefix variations so any Nginx proxy_pass style works flawlessly
	r.Route("/api/v1", registerRoutes)
	r.Route("/v1", registerRoutes)
	r.Route("/api", registerRoutes)
	r.Group(registerRoutes)

	return r
}
