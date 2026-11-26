package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/service"
	"business-scraping-service/backend/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type JobHandler struct {
	jobSvc *service.JobService
}

func NewJobHandler(jobSvc *service.JobService) *JobHandler {
	return &JobHandler{jobSvc: jobSvc}
}

func (h *JobHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	job, err := h.jobSvc.CreateJob(r.Context(), req)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondJSON(w, http.StatusCreated, job)
}

func (h *JobHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	jobs, total, err := h.jobSvc.ListJobs(r.Context(), page, pageSize)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to list jobs")
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"items":       jobs,
		"total_count": total,
		"page":        page,
		"page_size":   pageSize,
	})
}

func (h *JobHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid job ID")
		return
	}

	job, err := h.jobSvc.GetJob(r.Context(), id)
	if err != nil || job == nil {
		utils.RespondError(w, http.StatusNotFound, "Job not found")
		return
	}

	utils.RespondJSON(w, http.StatusOK, job)
}
