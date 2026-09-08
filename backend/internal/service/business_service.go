package service

import (
	"context"

	"github.com/google/uuid"
	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"
)

type BusinessService struct {
	businessRepo *repository.BusinessRepository
}

func NewBusinessService(businessRepo *repository.BusinessRepository) *BusinessService {
	return &BusinessService{businessRepo: businessRepo}
}

func (s *BusinessService) List(ctx context.Context, filter model.BusinessFilter) (*model.BusinessListResponse, error) {
	return s.businessRepo.List(ctx, filter)
}

func (s *BusinessService) GetByID(ctx context.Context, id uuid.UUID) (*model.BusinessLead, error) {
	return s.businessRepo.GetByID(ctx, id)
}

func (s *BusinessService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.businessRepo.Delete(ctx, id)
}
