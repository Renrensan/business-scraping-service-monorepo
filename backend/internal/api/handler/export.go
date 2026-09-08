package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/service"
	"business-scraping-service/backend/internal/utils"
)

type ExportHandler struct {
	exportSvc *service.ExportService
}

func NewExportHandler(exportSvc *service.ExportService) *ExportHandler {
	return &ExportHandler{exportSvc: exportSvc}
}

func (h *ExportHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := model.BusinessFilter{
		Search:   q.Get("search"),
		Category: q.Get("category"),
		City:     q.Get("city"),
	}

	if hasEmailStr := q.Get("has_email"); hasEmailStr != "" {
		hasEmail := hasEmailStr == "true"
		filter.HasEmail = &hasEmail
	}
	if ratingStr := q.Get("min_rating"); ratingStr != "" {
		if val, err := strconv.ParseFloat(ratingStr, 64); err == nil {
			filter.MinRating = &val
		}
	}

	csvBytes, _, err := h.exportSvc.ExportCSV(r.Context(), filter)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to generate CSV: "+err.Error())
		return
	}

	filename := fmt.Sprintf("business_leads_%s.csv", time.Now().Format("2006-01-02_150405"))

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(csvBytes)
}
