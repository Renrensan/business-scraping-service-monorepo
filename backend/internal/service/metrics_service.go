package service

import (
	"context"
	"fmt"

	"business-scraping-service/backend/internal/engine"
	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"
)

type MetricsService struct {
	businessRepo *repository.BusinessRepository
	scheduler    *engine.Scheduler
}

func NewMetricsService(businessRepo *repository.BusinessRepository, scheduler *engine.Scheduler) *MetricsService {
	return &MetricsService{
		businessRepo: businessRepo,
		scheduler:    scheduler,
	}
}

func (s *MetricsService) GetOverview(ctx context.Context) (*model.OverviewMetrics, error) {
	metrics, err := s.businessRepo.GetOverviewMetrics(ctx)
	if err != nil {
		return nil, err
	}

	if s.scheduler != nil {
		info := s.scheduler.GetScheduleInfo()
		metrics.NextScrapeAt = &info.NextRunAt
		metrics.CountdownSeconds = info.CountdownSeconds
		metrics.AutoScheduleEnabled = info.AutoEnabled
		metrics.ScheduleInterval = fmt.Sprintf("Every %dh (Daily Batch)", info.IntervalHours)
	}

	return metrics, nil
}

func (s *MetricsService) GetCategoryStats(ctx context.Context, limit int) ([]*model.CategoryStat, error) {
	return s.businessRepo.GetCategoryStats(ctx, limit)
}

func (s *MetricsService) GetLocationStats(ctx context.Context, limit int) ([]*model.LocationStat, error) {
	return s.businessRepo.GetLocationStats(ctx, limit)
}
