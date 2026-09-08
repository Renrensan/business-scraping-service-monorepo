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
