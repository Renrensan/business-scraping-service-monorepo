package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/service"
	"business-scraping-service/backend/internal/utils"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type KeywordHandler struct {
	keywordSvc *service.KeywordService
}

func NewKeywordHandler(keywordSvc *service.KeywordService) *KeywordHandler {
	return &KeywordHandler{keywordSvc: keywordSvc}
}

func (h *KeywordHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize <= 0 {
		pageSize = 50
	}

	filter := model.ListKeywordFilter{
		Search:   r.URL.Query().Get("search"),
		Status:   r.URL.Query().Get("status"),
		Page:     page,
		PageSize: pageSize,
	}

	resp, err := h.keywordSvc.List(r.Context(), filter)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to list keywords: "+err.Error())
		return
	}

	utils.RespondJSON(w, http.StatusOK, resp)
}

func (h *KeywordHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateKeywordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if strings.TrimSpace(req.Term) != "" || len(req.AreaIDs) > 0 {
		res, err := h.keywordSvc.CreateAreaKeywords(r.Context(), req.Term, req.IncludedType, req.AreaIDs)
		if err != nil {
			utils.RespondError(w, http.StatusBadRequest, "Failed to create keywords: "+err.Error())
			return
		}
		utils.RespondSuccess(w, http.StatusCreated, "Keywords created", map[string]interface{}{
			"count":    res.Created,
			"existing": res.Existing,
			"keywords": res.Keywords,
		})
		return
	}

	if len(req.Keywords) > 0 {
		createdCount, err := h.keywordSvc.CreateBulk(r.Context(), req.Keywords)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "Failed to create keywords: "+err.Error())
			return
		}
		utils.RespondSuccess(w, http.StatusCreated, "Keywords created", map[string]interface{}{
			"count": createdCount,
		})
		return
	}

	if strings.TrimSpace(req.Keyword) != "" {
		kw, err := h.keywordSvc.Create(r.Context(), req.Keyword)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, "Failed to create keyword: "+err.Error())
			return
		}
		utils.RespondJSON(w, http.StatusCreated, kw)
		return
	}

	utils.RespondError(w, http.StatusBadRequest, "Keyword or keywords array is required")
}

func (h *KeywordHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid keyword ID")
		return
	}

	var opts service.TriggerScrapeOptions
	opts.MaxPages = 1
	opts.EnrichContacts = true

	if r.Body != nil && r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&opts)
	}

	result, err := h.keywordSvc.TriggerKeywordScrape(r.Context(), id, opts)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrKeywordBusy) {
			status = http.StatusConflict
		}
		utils.RespondError(w, status, "Failed to trigger scrape: "+err.Error())
		return
	}

	utils.RespondJSON(w, http.StatusOK, result)
}

func (h *KeywordHandler) Reset(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid keyword ID")
		return
	}

	if err := h.keywordSvc.ResetCursor(r.Context(), id); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to reset cursor: "+err.Error())
		return
	}

	utils.RespondSuccess(w, http.StatusOK, "Keyword cursor reset to Page 1", nil)
}

func (h *KeywordHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid keyword ID")
		return
	}

	var req struct {
		Keyword  string `json:"keyword"`
		IsActive *bool  `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	kw, err := h.keywordSvc.Update(r.Context(), id, req.Keyword, isActive)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to update keyword: "+err.Error())
		return
	}

	utils.RespondJSON(w, http.StatusOK, kw)
}

func (h *KeywordHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid keyword ID")
		return
	}

	if err := h.keywordSvc.Delete(r.Context(), id); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to delete keyword: "+err.Error())
		return
	}

	utils.RespondSuccess(w, http.StatusOK, "Keyword deleted", nil)
}

func (h *KeywordHandler) SuggestType(w http.ResponseWriter, r *http.Request) {
	utils.RespondJSON(w, http.StatusOK, map[string]string{
		"included_type": h.keywordSvc.SuggestType(r.URL.Query().Get("term")),
	})
}
