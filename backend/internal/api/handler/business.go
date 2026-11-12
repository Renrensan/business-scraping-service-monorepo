package handler

import (
	"encoding/json"
	"net/http"
	"strings"

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
	filter := parseBusinessFilter(r.URL.Query())

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

// PatchStatus sets a lead's status to cold, warm or hot (which also marks it contacted).
func (h *BusinessHandler) PatchStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid business ID")
		return
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	status := model.LeadStatus(strings.ToLower(strings.TrimSpace(body.Status)))
	if !status.Valid() {
		utils.RespondError(w, http.StatusBadRequest, "Status must be one of: cold, warm, hot")
		return
	}

	found, err := h.businessSvc.SetLeadStatus(r.Context(), id, status)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update lead status")
		return
	}
	if !found {
		utils.RespondError(w, http.StatusNotFound, "Business lead not found")
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

