package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"business-scraping-service/backend/internal/engine"
	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"
)

type JobService struct {
	jobRepo    *repository.JobRepository
	workerPool *engine.WorkerPool
}

func NewJobService(jobRepo *repository.JobRepository, workerPool *engine.WorkerPool) *JobService {
	return &JobService{
		jobRepo:    jobRepo,
		workerPool: workerPool,
	}
}

func (s *JobService) CreateJob(ctx context.Context, req model.CreateJobRequest) (*model.ScrapeJob, error) {
	if req.Query == "" {
		return nil, fmt.Errorf("search query is required")
	}
	if req.MaxResults <= 0 {
		req.MaxResults = 50
	}
	if req.MaxResults > 1000 {
		req.MaxResults = 1000
	}

	meta := make(map[string]interface{})
	if req.Engine != "" {
		meta["engine"] = req.Engine
	}
	if req.ApiKey != "" {
		meta["api_key"] = req.ApiKey
	}

	job := &model.ScrapeJob{
		ID:             uuid.New(),
		Status:         model.JobStatusPending,
		Query:          req.Query,
		Location:       req.Location,
		Category:       req.Category,
		MaxResults:     req.MaxResults,
		EnrichContacts: req.EnrichContacts,
		Metadata:       meta,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := s.jobRepo.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to insert scrape job: %w", err)
	}

	// Dispatch asynchronously to worker pool
	s.workerPool.SubmitJob(job)

	return job, nil
}

func (s *JobService) GetJob(ctx context.Context, id uuid.UUID) (*model.ScrapeJob, error) {
	return s.jobRepo.GetByID(ctx, id)
}

func (s *JobService) ListJobs(ctx context.Context, page, pageSize int) ([]*model.ScrapeJob, int64, error) {
	return s.jobRepo.List(ctx, page, pageSize)
}
