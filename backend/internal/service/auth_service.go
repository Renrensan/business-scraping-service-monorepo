package service

import (
	"context"
	"fmt"
	"time"

	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"
	"business-scraping-service/backend/internal/session"
	"business-scraping-service/backend/internal/utils"
	"github.com/google/uuid"
)

type AuthService struct {
	adminRepo    *repository.AdminRepository
	sessionStore session.Store
}

func NewAuthService(adminRepo *repository.AdminRepository, sessionStore session.Store) *AuthService {
	return &AuthService{
		adminRepo:    adminRepo,
		sessionStore: sessionStore,
	}
}

func (s *AuthService) Login(ctx context.Context, usernameOrEmail, password string) (*model.AdminUser, string, error) {
	admin, err := s.adminRepo.GetByUsernameOrEmail(ctx, usernameOrEmail)
	if err != nil {
		return nil, "", fmt.Errorf("database query error: %w", err)
	}
	if admin == nil || !admin.IsActive {
		return nil, "", fmt.Errorf("invalid credentials or inactive account")
	}

	if !utils.CheckPasswordHash(password, admin.PasswordHash) {
		return nil, "", fmt.Errorf("invalid credentials")
	}

	sessionData := model.SessionData{
		AdminID:      admin.ID,
		Username:     admin.Username,
		Email:        admin.Email,
		CreatedAt:    time.Now().UTC(),
		LastActiveAt: time.Now().UTC(),
	}

	token, err := s.sessionStore.CreateSession(ctx, sessionData)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create session: %w", err)
	}

	return admin, token, nil
}

func (s *AuthService) ValidateSession(ctx context.Context, token string) (*model.SessionData, error) {
	data, err := s.sessionStore.GetSession(ctx, token)
	if err != nil || data == nil {
		return nil, err
	}
	_ = s.sessionStore.TouchSession(ctx, token)
	return data, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.sessionStore.DeleteSession(ctx, token)
}

func (s *AuthService) ChangePassword(ctx context.Context, adminID uuid.UUID, currentPassword, newPassword string) error {
	admin, err := s.adminRepo.GetByID(ctx, adminID)
	if err != nil || admin == nil {
		return fmt.Errorf("admin user not found")
	}

	if !utils.CheckPasswordHash(currentPassword, admin.PasswordHash) {
		return fmt.Errorf("current password does not match")
	}

	newHash, err := utils.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	return s.adminRepo.UpdatePassword(ctx, adminID, newHash)
}
