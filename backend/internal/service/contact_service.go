package service

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"business-scraping-service/backend/internal/apperr"
	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"
	"business-scraping-service/backend/internal/utils"
)

type ContactService struct {
	repo *repository.ContactRepository
}

func NewContactService(repo *repository.ContactRepository) *ContactService {
	return &ContactService{repo: repo}
}

const (
	maxContactField = 200
	maxContactNote  = 2000
)

// cleanContact validates a contact and normalises it: the phone is stored in international
// format (+62...) so WhatsApp links work, the email lowercased.
func cleanContact(in model.LeadContactInput) (model.LeadContactInput, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Role = strings.TrimSpace(in.Role)
	in.Phone = strings.TrimSpace(in.Phone)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Note = strings.TrimSpace(in.Note)

	if in.Name == "" {
		return in, apperr.Invalid("name is required")
	}
	for field, v := range map[string]string{"name": in.Name, "role": in.Role, "phone": in.Phone, "email": in.Email} {
		if utf8.RuneCountInString(v) > maxContactField {
			return in, apperr.Invalid("%s is too long (max %d characters)", field, maxContactField)
		}
	}
	if utf8.RuneCountInString(in.Note) > maxContactNote {
		return in, apperr.Invalid("note is too long (max %d characters)", maxContactNote)
	}
	if in.Phone != "" {
		number, _, ok := utils.ToWhatsAppNumber(in.Phone)
		if !ok {
			return in, apperr.Invalid("phone %q doesn't look like a phone number", in.Phone)
		}
		in.Phone = number
	}
	if in.Email != "" {
		addr, err := mail.ParseAddress(in.Email)
		if err != nil || addr.Address != in.Email || !strings.Contains(in.Email[strings.LastIndex(in.Email, "@"):], ".") {
			return in, apperr.Invalid("email %q doesn't look like an email address", in.Email)
		}
	}
	return in, nil
}

func (s *ContactService) List(ctx context.Context, leadID uuid.UUID) ([]model.LeadContact, error) {
	return s.repo.List(ctx, leadID)
}

func (s *ContactService) Create(ctx context.Context, leadID uuid.UUID, in model.LeadContactInput) (*model.LeadContact, error) {
	clean, err := cleanContact(in)
	if err != nil {
		return nil, err
	}
	c, err := s.repo.Create(ctx, leadID, clean)
	if err != nil {
		return nil, fmt.Errorf("adding contact: %w", err)
	}
	return c, nil
}

func (s *ContactService) Update(ctx context.Context, leadID, contactID uuid.UUID, in model.LeadContactInput) (*model.LeadContact, error) {
	clean, err := cleanContact(in)
	if err != nil {
		return nil, err
	}
	c, err := s.repo.Update(ctx, leadID, contactID, clean)
	if err != nil {
		return nil, fmt.Errorf("updating contact: %w", err)
	}
	return c, nil
}

func (s *ContactService) Delete(ctx context.Context, leadID, contactID uuid.UUID) error {
	if err := s.repo.Delete(ctx, leadID, contactID); err != nil {
		return fmt.Errorf("deleting contact: %w", err)
	}
	return nil
}
