package service

import (
	"context"

	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"
)

type MetricsService struct {
	businessRepo *repository.BusinessRepository
}

func NewMetricsService(businessRepo *repository.BusinessRepository) *MetricsService {
	return &MetricsService{businessRepo: businessRepo}
}

func (s *MetricsService) GetOverview(ctx context.Context) (*model.OverviewMetrics, error) {
	return s.businessRepo.GetOverviewMetrics(ctx)
}

func (s *MetricsService) GetCategoryStats(ctx context.Context, limit int) ([]*model.CategoryStat, error) {
	return s.businessRepo.GetCategoryStats(ctx, limit)
}

func (s *MetricsService) GetLocationStats(ctx context.Context, limit int) ([]*model.LocationStat, error) {
	return s.businessRepo.GetLocationStats(ctx, limit)
}
