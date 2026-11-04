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
	if req.MaxResults <= 0 {
		req.MaxResults = 50
	}
	if req.MaxResults > 1000 {
		req.MaxResults = 1000
	}
	if req.MaxReviewCountForNew <= 0 {
		req.MaxReviewCountForNew = 30
	}

	meta := make(map[string]interface{})
	if req.Engine != "" {
		meta["engine"] = req.Engine
	}
	if req.ApiKey != "" {
		meta["api_key"] = req.ApiKey
	}
	// Recency Filter
	recency := "any"
	if req.RecencyFilter != "" {
		recency = req.RecencyFilter
	} else if req.OnlyNewBusinesses {
		recency = "only_new"
	}
	meta["recency_filter"] = recency
	if recency == "only_new" {
		meta["only_new_businesses"] = true
		meta["max_review_count_for_new"] = req.MaxReviewCountForNew
	}

	// Website Filter
	webFilter := "any"
	if req.WebsiteFilter != "" {
		webFilter = req.WebsiteFilter
	} else if req.RequireWebsite {
		webFilter = "require_website"
	}
	meta["website_filter"] = webFilter
	if webFilter == "require_website" {
		meta["require_website"] = true
	}

	var queries []string
	if len(req.Queries) > 0 {
		queries = req.Queries
	} else if req.Query != "" {
		queries = []string{req.Query}
	} else {
		return nil, fmt.Errorf("at least one search query is required")
	}

	var locations []string
	if len(req.Locations) > 0 {
		locations = req.Locations
	} else if req.Location != "" {
		locations = []string{req.Location}
	} else {
		locations = []string{"Bali, Indonesia"}
	}

	var firstJob *model.ScrapeJob

	for _, loc := range locations {
		for i, q := range queries {
			cat := req.Category
			if i < len(req.Categories) && req.Categories[i] != "" {
				cat = req.Categories[i]
			}

			jobMeta := make(map[string]interface{})
			for k, v := range meta {
				jobMeta[k] = v
			}

			job := &model.ScrapeJob{
				ID:             uuid.New(),
				Status:         model.JobStatusPending,
				Query:          q,
				Location:       loc,
				Category:       cat,
				MaxResults:     req.MaxResults,
				EnrichContacts: req.EnrichContacts,
				Metadata:       jobMeta,
				CreatedAt:      time.Now().UTC(),
				UpdatedAt:      time.Now().UTC(),
			}

			if err := s.jobRepo.Create(ctx, job); err != nil {
				return nil, fmt.Errorf("failed to insert scrape job: %w", err)
			}

			s.workerPool.SubmitJob(job)

			if firstJob == nil {
				firstJob = job
			}
		}
	}

	return firstJob, nil
}

func (s *JobService) GetJob(ctx context.Context, id uuid.UUID) (*model.ScrapeJob, error) {
	return s.jobRepo.GetByID(ctx, id)
}

func (s *JobService) ListJobs(ctx context.Context, page, pageSize int) ([]*model.ScrapeJob, int64, error) {
	return s.jobRepo.List(ctx, page, pageSize)
}
