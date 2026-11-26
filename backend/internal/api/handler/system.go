package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"business-scraping-service/backend/internal/service"
	"business-scraping-service/backend/internal/utils"
)

type SystemHandler struct {
	systemSvc *service.SystemService
}

func NewSystemHandler(systemSvc *service.SystemService) *SystemHandler {
	return &SystemHandler{systemSvc: systemSvc}
}

func (h *SystemHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	health := h.systemSvc.GetHealth(r.Context())
	status := http.StatusOK
	if health.Status == "degraded" {
		status = http.StatusServiceUnavailable
	}
	utils.RespondJSON(w, status, health)
}

func (h *SystemHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings := h.systemSvc.GetSettings()
	utils.RespondJSON(w, http.StatusOK, settings)
}

type UpdateAPIKeyRequest struct {
	ApiKey string `json:"api_key"`
}

func (h *SystemHandler) UpdateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req UpdateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := h.systemSvc.UpdateGoogleAPIKey(r.Context(), req.ApiKey); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "API Key updated successfully", h.systemSvc.GetSettings())
}

func (h *SystemHandler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	sched := h.systemSvc.GetSchedule()
	utils.RespondJSON(w, http.StatusOK, sched)
}

type UpdateScheduleRequest struct {
	IntervalHours        int      `json:"interval_hours"`
	AutoEnabled          bool     `json:"auto_enabled"`
	TargetQuery          string   `json:"target_query"`
	TargetLocation       string   `json:"target_location"`
	TargetCategory       string   `json:"target_category"`
	TargetQueries        []string `json:"target_queries"`
	TargetLocations      []string `json:"target_locations"`
	TargetCategories     []string `json:"target_categories"`
	OnlyNewBusinesses    bool     `json:"only_new_businesses"`
	RequireWebsite       bool     `json:"require_website"`
	WebsiteFilter        string   `json:"website_filter"`
	RecencyFilter        string   `json:"recency_filter"`
	MaxReviewCountForNew int      `json:"max_review_count_for_new"`
	MaxPagesPerRun       int      `json:"max_pages_per_run"`
	PageSize             int      `json:"page_size"`
	MaxResults           int      `json:"max_results"`
	EnrichContacts       bool     `json:"enrich_contacts"`
	Engine               string   `json:"engine"`
}

func (h *SystemHandler) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	var req UpdateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	queries := req.TargetQueries
	if len(queries) == 0 && req.TargetQuery != "" {
		queries = []string{req.TargetQuery}
	}

	locations := req.TargetLocations
	if len(locations) == 0 && req.TargetLocation != "" {
		locations = []string{req.TargetLocation}
	}

	categories := req.TargetCategories
	if len(categories) == 0 && req.TargetCategory != "" {
		categories = []string{req.TargetCategory}
	}

	sched := h.systemSvc.UpdateSchedule(
		req.IntervalHours,
		req.AutoEnabled,
		queries,
		locations,
		categories,
		req.OnlyNewBusinesses,
		req.RequireWebsite,
		req.WebsiteFilter,
		req.RecencyFilter,
		req.MaxReviewCountForNew,
		req.MaxPagesPerRun,
		req.PageSize,
		req.MaxResults,
		req.EnrichContacts,
		req.Engine,
	)
	utils.RespondSuccess(w, http.StatusOK, "Schedule updated successfully", sched)
}

type TriggerScrapeRequest struct {
	Query                string   `json:"query"`
	Location             string   `json:"location"`
	Category             string   `json:"category"`
	Queries              []string `json:"queries"`
	Locations            []string `json:"locations"`
	Categories           []string `json:"categories"`
	MaxResults           int      `json:"max_results"`
	EnrichContacts       bool     `json:"enrich_contacts"`
	Engine               string   `json:"engine"`
	OnlyNewBusinesses    bool     `json:"only_new_businesses"`
	RequireWebsite       bool     `json:"require_website"`
	WebsiteFilter        string   `json:"website_filter"`
	RecencyFilter        string   `json:"recency_filter"`
	MaxReviewCountForNew int      `json:"max_review_count_for_new"`
}

func (h *SystemHandler) TriggerImmediateScrape(w http.ResponseWriter, r *http.Request) {
	var req TriggerScrapeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	queries := req.Queries
	if len(queries) == 0 && req.Query != "" {
		queries = []string{req.Query}
	}

	locations := req.Locations
	if len(locations) == 0 && req.Location != "" {
		locations = []string{req.Location}
	}

	categories := req.Categories
	if len(categories) == 0 && req.Category != "" {
		categories = []string{req.Category}
	}

	job, err := h.systemSvc.TriggerImmediateScrape(
		r.Context(),
		queries,
		locations,
		categories,
		req.MaxResults,
		req.EnrichContacts,
		req.Engine,
		req.OnlyNewBusinesses,
		req.RequireWebsite,
		req.WebsiteFilter,
		req.RecencyFilter,
		req.MaxReviewCountForNew,
	)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondSuccess(w, http.StatusOK, "Immediate ingestion triggered", job)
}

type UpdatePlacesBudgetRequest struct {
	DailyBudget int `json:"daily_budget"`
}

func (h *SystemHandler) UpdatePlacesBudget(w http.ResponseWriter, r *http.Request) {
	var req UpdatePlacesBudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if err := h.systemSvc.SetPlacesBudget(r.Context(), req.DailyBudget); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "Daily budget updated", h.systemSvc.GetSettings())
}

func (h *SystemHandler) GetScrapeHealth(w http.ResponseWriter, r *http.Request) {
	health, err := h.systemSvc.ScrapeHealth(r.Context())
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to load scrape health: "+err.Error())
		return
	}
	utils.RespondJSON(w, http.StatusOK, health)
}
