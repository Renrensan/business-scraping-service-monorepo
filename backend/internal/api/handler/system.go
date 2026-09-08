package handler

import (
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
