package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"
	"business-scraping-service/backend/internal/service"
	"business-scraping-service/backend/internal/utils"
)

// AdminHandler manages admin accounts and serves the audit log.
type AdminHandler struct {
	svc   *service.AdminService
	audit *repository.AuditRepository
}

func NewAdminHandler(svc *service.AdminService, audit *repository.AuditRepository) *AdminHandler {
	return &AdminHandler{svc: svc, audit: audit}
}

func (h *AdminHandler) List(w http.ResponseWriter, r *http.Request) {
	admins, err := h.svc.List(r.Context())
	if err != nil {
		respondErr(w, r, err, "Failed to load admins")
		return
	}
	utils.RespondJSON(w, http.StatusOK, admins)
}

func (h *AdminHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in service.CreateAdminInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	a, err := h.svc.Create(r.Context(), in)
	if err != nil {
		respondErr(w, r, err, "Failed to add admin")
		return
	}
	utils.RespondSuccess(w, http.StatusCreated, "Admin added", a)
}

func (h *AdminHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid admin ID")
		return
	}
	var in service.UpdateAdminInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	actor := actingAdmin(r)
	if actor == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Session required")
		return
	}
	a, err := h.svc.Update(r.Context(), *actor, id, in)
	if err != nil {
		respondErr(w, r, err, "Failed to update admin")
		return
	}
	utils.RespondSuccess(w, http.StatusOK, "Admin updated", a)
}

// Audit lists audit entries: ?admin_id, entity_type, entity_id, action, search, failed=true, page, page_size.
func (h *AdminHandler) Audit(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := model.AuditFilter{
		EntityType: q.Get("entity_type"),
		EntityID:   q.Get("entity_id"),
		Action:     q.Get("action"),
		Search:     q.Get("search"),
		FailedOnly: q.Get("failed") == "true",
	}
	f.Page, _ = strconv.Atoi(q.Get("page"))          // 0 falls back to page 1
	f.PageSize, _ = strconv.Atoi(q.Get("page_size")) // 0 falls back to the default
	if v := q.Get("admin_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			utils.RespondError(w, http.StatusBadRequest, "Invalid admin_id")
			return
		}
		f.AdminID = &id
	}
	res, err := h.audit.List(r.Context(), f)
	if err != nil {
		respondErr(w, r, err, "Failed to load the audit log")
		return
	}
	utils.RespondJSON(w, http.StatusOK, res)
}
