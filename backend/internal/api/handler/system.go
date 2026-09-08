package handler

import (
	"encoding/json"
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

	h.systemSvc.UpdateGoogleAPIKey(req.ApiKey)
	utils.RespondSuccess(w, http.StatusOK, "API Key updated successfully", h.systemSvc.GetSettings())
}

func (h *SystemHandler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	sched := h.systemSvc.GetSchedule()
	utils.RespondJSON(w, http.StatusOK, sched)
}

type UpdateScheduleRequest struct {
	IntervalHours  int    `json:"interval_hours"`
	AutoEnabled    bool   `json:"auto_enabled"`
	TargetQuery    string `json:"target_query"`
	TargetLocation string `json:"target_location"`
}

func (h *SystemHandler) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	var req UpdateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	sched := h.systemSvc.UpdateSchedule(req.IntervalHours, req.AutoEnabled, req.TargetQuery, req.TargetLocation)
	utils.RespondSuccess(w, http.StatusOK, "Schedule updated successfully", sched)
}

type TriggerScrapeRequest struct {
	Query      string `json:"query"`
	Location   string `json:"location"`
	MaxResults int    `json:"max_results"`
}

func (h *SystemHandler) TriggerImmediateScrape(w http.ResponseWriter, r *http.Request) {
	var req TriggerScrapeRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	job, err := h.systemSvc.TriggerImmediateScrape(r.Context(), req.Query, req.Location, req.MaxResults)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondSuccess(w, http.StatusOK, "Immediate ingestion triggered", job)
}
