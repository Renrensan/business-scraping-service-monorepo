package handler

import (
	"encoding/json"
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
		Keyword:   q.Get("keyword"),
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
	if hasWebsiteStr := q.Get("has_website"); hasWebsiteStr != "" {
		hasWebsite := hasWebsiteStr == "true"
		filter.HasWebsite = &hasWebsite
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
	if onlyNewStr := q.Get("only_new"); onlyNewStr != "" {
		onlyNew := onlyNewStr == "true"
		filter.OnlyNew = &onlyNew
	}
	if maxReviewsStr := q.Get("max_reviews"); maxReviewsStr != "" {
		if val, err := strconv.Atoi(maxReviewsStr); err == nil {
			filter.MaxReviews = &val
		}
	}
	if contactedStr := q.Get("contacted"); contactedStr != "" {
		contacted := contactedStr == "true"
		filter.Contacted = &contacted
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

func (h *BusinessHandler) PatchContacted(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid business ID")
		return
	}

	var body struct {
		Contacted bool `json:"contacted"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.businessSvc.SetContacted(r.Context(), id, body.Contacted); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update contacted status")
		return
	}

	utils.RespondSuccess(w, http.StatusOK, "Updated", nil)
}

func (h *BusinessHandler) GetKeywords(w http.ResponseWriter, r *http.Request) {
	keywords, err := h.businessSvc.GetScrapeKeywords(r.Context())
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to get keywords")
		return
	}
	utils.RespondJSON(w, http.StatusOK, keywords)
}

func (h *BusinessHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.businessSvc.GetDerivedCategories(r.Context())
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to get derived categories")
		return
	}
	utils.RespondJSON(w, http.StatusOK, categories)
}

