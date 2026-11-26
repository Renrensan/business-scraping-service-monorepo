package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"business-scraping-service/backend/internal/service"
	"business-scraping-service/backend/internal/utils"
)

type ScoringHandler struct {
	svc *service.ScoringService
}

func NewScoringHandler(svc *service.ScoringService) *ScoringHandler {
	return &ScoringHandler{svc: svc}
}

func (h *ScoringHandler) Options(w http.ResponseWriter, r *http.Request) {
	utils.RespondJSON(w, http.StatusOK, h.svc.Options())
}

func (h *ScoringHandler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListProfiles(r.Context())
	if err != nil {
		respondErr(w, r, err, "Failed to load scoring profiles")
		return
	}
	utils.RespondJSON(w, http.StatusOK, list)
}

func (h *ScoringHandler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	var in service.ProfileInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	p, err := h.svc.CreateProfile(r.Context(), in)
	if err != nil {
		respondErr(w, r, err, "Failed to create scoring profile")
		return
	}
	utils.RespondSuccess(w, http.StatusCreated, "Scoring profile created", p)
}

func (h *ScoringHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid profile ID")
		return
	}
	var in service.ProfileInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	p, err := h.svc.UpdateProfile(r.Context(), id, in)
	if err != nil {
		respondErr(w, r, err, "Failed to update scoring profile")
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "Scoring profile updated", p)
}

func (h *ScoringHandler) DeleteProfile(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid profile ID")
		return
	}
	if err := h.svc.DeleteProfile(r.Context(), id); err != nil {
		respondErr(w, r, err, "Failed to delete scoring profile")
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "Scoring profile deleted", nil)
}

func (h *ScoringHandler) Rescore(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Rescore(r.Context()); err != nil {
		respondErr(w, r, err, "Failed to rescore leads")
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "All leads rescored", nil)
}

func (h *ScoringHandler) SetLeadGrouping(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid business ID")
		return
	}
	var in service.LeadGroupingInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if err := h.svc.SetLeadGrouping(r.Context(), id, in); err != nil {
		respondErr(w, r, err, "Failed to update group")
		return
	}
	scores, err := h.svc.ScoresForLead(r.Context(), id)
	if err != nil {
		slog.Error("saved, but failed to reload lead scores", "lead_id", id, "error", err)
		scores = nil
	}
	utils.RespondSuccess(w, http.StatusOK, "Group updated", scores)
}

func (h *ScoringHandler) SetScoreOverride(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid business ID")
		return
	}
	profileID, err := uuid.Parse(chi.URLParam(r, "profileID"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid profile ID")
		return
	}
	var in service.ScoreOverrideInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if err := h.svc.SetScoreOverride(r.Context(), id, profileID, in); err != nil {
		respondErr(w, r, err, "Failed to update tier")
		return
	}
	scores, err := h.svc.ScoresForLead(r.Context(), id)
	if err != nil {
		slog.Error("saved, but failed to reload lead scores", "lead_id", id, "error", err)
		scores = nil
	}
	utils.RespondSuccess(w, http.StatusOK, "Tier updated", scores)
}
