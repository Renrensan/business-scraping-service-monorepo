package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"business-scraping-service/backend/internal/service"
	"business-scraping-service/backend/internal/utils"
)

type AreaHandler struct {
	keywordSvc *service.KeywordService
}

func NewAreaHandler(keywordSvc *service.KeywordService) *AreaHandler {
	return &AreaHandler{keywordSvc: keywordSvc}
}

func (h *AreaHandler) List(w http.ResponseWriter, r *http.Request) {
	areas, err := h.keywordSvc.ListAreas(r.Context())
	if err != nil {
		respondErr(w, r, err, "Failed to list areas")
		return
	}
	utils.RespondJSON(w, http.StatusOK, areas)
}

func (h *AreaHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req service.AddAreaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	area, err := h.keywordSvc.AddArea(r.Context(), req)
	if err != nil {
		respondErr(w, r, err, "Failed to add area")
		return
	}
	utils.RespondSuccess(w, http.StatusCreated, "Area added", area)
}

func (h *AreaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid area ID")
		return
	}
	if err := h.keywordSvc.DeleteArea(r.Context(), id); err != nil {
		respondErr(w, r, err, "Failed to delete area")
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "Area deleted", nil)
}
