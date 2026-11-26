package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/service"
	"business-scraping-service/backend/internal/utils"
)

type ContactHandler struct {
	svc *service.ContactService
}

func NewContactHandler(svc *service.ContactService) *ContactHandler {
	return &ContactHandler{svc: svc}
}

func parseIDs(w http.ResponseWriter, r *http.Request, withContact bool) (leadID, contactID uuid.UUID, ok bool) {
	leadID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid business ID")
		return leadID, contactID, false
	}
	if withContact {
		contactID, err = uuid.Parse(chi.URLParam(r, "contactID"))
		if err != nil {
			utils.RespondError(w, http.StatusBadRequest, "Invalid contact ID")
			return leadID, contactID, false
		}
	}
	return leadID, contactID, true
}

func decodeContact(w http.ResponseWriter, r *http.Request) (model.LeadContactInput, bool) {
	var in model.LeadContactInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return in, false
	}
	return in, true
}

func (h *ContactHandler) List(w http.ResponseWriter, r *http.Request) {
	leadID, _, ok := parseIDs(w, r, false)
	if !ok {
		return
	}
	contacts, err := h.svc.List(r.Context(), leadID)
	if err != nil {
		respondErr(w, r, err, "Failed to load contacts")
		return
	}
	utils.RespondJSON(w, http.StatusOK, contacts)
}

func (h *ContactHandler) Create(w http.ResponseWriter, r *http.Request) {
	leadID, _, ok := parseIDs(w, r, false)
	if !ok {
		return
	}
	in, ok := decodeContact(w, r)
	if !ok {
		return
	}
	c, err := h.svc.Create(r.Context(), leadID, in)
	if err != nil {
		respondErr(w, r, err, "Failed to add contact")
		return
	}
	utils.RespondSuccess(w, http.StatusCreated, "Contact added", c)
}

func (h *ContactHandler) Update(w http.ResponseWriter, r *http.Request) {
	leadID, contactID, ok := parseIDs(w, r, true)
	if !ok {
		return
	}
	in, ok := decodeContact(w, r)
	if !ok {
		return
	}
	c, err := h.svc.Update(r.Context(), leadID, contactID, in)
	if err != nil {
		respondErr(w, r, err, "Failed to update contact")
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "Contact updated", c)
}

func (h *ContactHandler) Delete(w http.ResponseWriter, r *http.Request) {
	leadID, contactID, ok := parseIDs(w, r, true)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), leadID, contactID); err != nil {
		respondErr(w, r, err, "Failed to delete contact")
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "Contact deleted", nil)
}
