package service

import (
	"context"

	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"
	"github.com/google/uuid"
)

type BusinessService struct {
	businessRepo *repository.BusinessRepository
	scoringRepo  *repository.ScoringRepository
	contactRepo  *repository.ContactRepository
}

func NewBusinessService(businessRepo *repository.BusinessRepository, scoringRepo *repository.ScoringRepository, contactRepo *repository.ContactRepository) *BusinessService {
	return &BusinessService{businessRepo: businessRepo, scoringRepo: scoringRepo, contactRepo: contactRepo}
}

func (s *BusinessService) List(ctx context.Context, filter model.BusinessFilter) (*model.BusinessListResponse, error) {
	return s.businessRepo.List(ctx, filter)
}

func (s *BusinessService) GetByID(ctx context.Context, id uuid.UUID) (*model.BusinessLead, error) {
	lead, err := s.businessRepo.GetByID(ctx, id)
	if err != nil || lead == nil || s.scoringRepo == nil {
		return lead, err
	}
	scores, err := s.scoringRepo.ScoresForLead(ctx, id)
	if err != nil {
		return nil, err
	}
	lead.Scores = scores
	if s.contactRepo != nil {
		contacts, err := s.contactRepo.List(ctx, id)
		if err != nil {
			return nil, err
		}
		lead.Contacts = contacts
		lead.ContactCount = len(contacts)
	}
	return lead, nil
}

func (s *BusinessService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.businessRepo.Delete(ctx, id)
}

func (s *BusinessService) SetContacted(ctx context.Context, id uuid.UUID, contacted bool) error {
	return s.businessRepo.SetContacted(ctx, id, contacted)
}

func (s *BusinessService) SetLeadStatus(ctx context.Context, id uuid.UUID, status model.LeadStatus) (bool, error) {
	return s.businessRepo.SetLeadStatus(ctx, id, status)
}

func (s *BusinessService) GetScrapeKeywords(ctx context.Context) ([]string, error) {
	return s.businessRepo.GetScrapeKeywords(ctx)
}

func (s *BusinessService) GetDerivedCategories(ctx context.Context) ([]string, error) {
	return s.businessRepo.GetDerivedCategories(ctx)
}
