package handler

import (
	"net/http"
	"strconv"

	"business-scraping-service/backend/internal/service"
	"business-scraping-service/backend/internal/utils"
)

type MetricsHandler struct {
	metricsSvc *service.MetricsService
}

func NewMetricsHandler(metricsSvc *service.MetricsService) *MetricsHandler {
	return &MetricsHandler{metricsSvc: metricsSvc}
}

func (h *MetricsHandler) GetOverview(w http.ResponseWriter, r *http.Request) {
	overview, err := h.metricsSvc.GetOverview(r.Context())
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to get overview metrics")
		return
	}

	utils.RespondJSON(w, http.StatusOK, overview)
}

func (h *MetricsHandler) GetCategoryStats(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	stats, err := h.metricsSvc.GetCategoryStats(r.Context(), limit)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to get category stats")
		return
	}

	utils.RespondJSON(w, http.StatusOK, stats)
}

func (h *MetricsHandler) GetLocationStats(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	stats, err := h.metricsSvc.GetLocationStats(r.Context(), limit)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to get location stats")
		return
	}

	utils.RespondJSON(w, http.StatusOK, stats)
}
