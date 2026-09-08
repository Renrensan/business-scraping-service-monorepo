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
		AllowedOrigins:   []string{cfg.FrontendURL, "http://localhost:5173", "http://localhost:3000", "http://127.0.0.1:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Root Healthcheck
	r.Get("/health", h.System.GetHealth)

	// API v1 Routes
	r.Route("/api/v1", func(api chi.Router) {
		// Public Auth
		api.Post("/auth/login", h.Auth.Login)
		api.Get("/system/health", h.System.GetHealth)

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

			// Businesses / Leads Database
			protected.Get("/businesses", h.Business.List)
			protected.Get("/businesses/{id}", h.Business.GetByID)
			protected.Delete("/businesses/{id}", h.Business.Delete)

			// Export
			protected.Get("/export/csv", h.Export.ExportCSV)

			// System Settings
			protected.Get("/system/settings", h.System.GetSettings)
			protected.Post("/system/api-key", h.System.UpdateAPIKey)
		})
	})

	return r
}
