package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/service"
	"business-scraping-service/backend/internal/utils"
)

type BusinessHandler struct {
	businessSvc *service.BusinessService
}

func NewBusinessHandler(businessSvc *service.BusinessService) *BusinessHandler {
	return &BusinessHandler{businessSvc: businessSvc}
}

func (h *BusinessHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))

	filter := model.BusinessFilter{
		Search:    q.Get("search"),
		Category:  q.Get("category"),
		City:      q.Get("city"),
		State:     q.Get("state"),
		SortBy:    q.Get("sort_by"),
		SortOrder: q.Get("sort_order"),
		Page:      page,
		PageSize:  pageSize,
	}

	if hasEmailStr := q.Get("has_email"); hasEmailStr != "" {
		hasEmail := hasEmailStr == "true"
		filter.HasEmail = &hasEmail
	}
	if hasPhoneStr := q.Get("has_phone"); hasPhoneStr != "" {
		hasPhone := hasPhoneStr == "true"
		filter.HasPhone = &hasPhone
	}
	if ratingStr := q.Get("min_rating"); ratingStr != "" {
		if val, err := strconv.ParseFloat(ratingStr, 64); err == nil {
			filter.MinRating = &val
		}
	}
	if jobIDStr := q.Get("job_id"); jobIDStr != "" {
		if id, err := uuid.Parse(jobIDStr); err == nil {
			filter.JobID = &id
		}
	}

	resp, err := h.businessSvc.List(r.Context(), filter)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to query businesses: "+err.Error())
		return
	}

	utils.RespondJSON(w, http.StatusOK, resp)
}

func (h *BusinessHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid business ID")
		return
	}

	lead, err := h.businessSvc.GetByID(r.Context(), id)
	if err != nil || lead == nil {
		utils.RespondError(w, http.StatusNotFound, "Business lead not found")
		return
	}

	utils.RespondJSON(w, http.StatusOK, lead)
}

func (h *BusinessHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid business ID")
		return
	}

	if err := h.businessSvc.Delete(r.Context(), id); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to delete lead")
		return
	}

	utils.RespondSuccess(w, http.StatusOK, "Lead deleted successfully", nil)
}
