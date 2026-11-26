package service

import (
	"context"
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"business-scraping-service/backend/internal/apperr"
	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"
	"business-scraping-service/backend/internal/session"
	"business-scraping-service/backend/internal/utils"
)

// MinPasswordLength applies to new admins, password resets and password changes.
const MinPasswordLength = 8

var (
	usernamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,49}$`)
	notUsernameChar = regexp.MustCompile(`[^a-z0-9._-]+`)
)

// usernameFromEmail derives a valid username from the part before "@"
// ("first.last+tag" -> "first.last-tag").
func usernameFromEmail(email string) string {
	u := notUsernameChar.ReplaceAllString(email[:strings.Index(email, "@")], "-")
	u = strings.TrimLeft(u, "._-")
	if len(u) > 50 {
		u = u[:50]
	}
	return u
}

// AdminService manages admin accounts. Every admin can manage the others; nobody can
// deactivate themselves, and the last active admin can't be deactivated.
type AdminService struct {
	repo     *repository.AdminRepository
	sessions session.Store
}

func NewAdminService(repo *repository.AdminRepository, sessions session.Store) *AdminService {
	return &AdminService{repo: repo, sessions: sessions}
}

func (s *AdminService) List(ctx context.Context) ([]model.AdminUser, error) {
	return s.repo.List(ctx)
}

// CreateAdminInput is a new admin. Username defaults to the part of the email before "@".
type CreateAdminInput struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func normalizeAdmin(in CreateAdminInput) (CreateAdminInput, error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Username = strings.ToLower(strings.TrimSpace(in.Username))
	if addr, err := mail.ParseAddress(in.Email); err != nil || addr.Address != in.Email {
		return in, apperr.Invalid("email %q doesn't look like an email address", in.Email)
	}
	if in.Username == "" {
		in.Username = usernameFromEmail(in.Email)
	}
	if !usernamePattern.MatchString(in.Username) {
		return in, apperr.Invalid("username may use lowercase letters, digits, '.', '_' and '-' (up to 50 characters)")
	}
	if err := checkPassword(in.Password); err != nil {
		return in, err
	}
	return in, nil
}

func checkPassword(p string) error {
	if len(p) < MinPasswordLength {
		return apperr.Invalid("password must be at least %d characters", MinPasswordLength)
	}
	return nil
}

func (s *AdminService) Create(ctx context.Context, in CreateAdminInput) (*model.AdminUser, error) {
	in, err := normalizeAdmin(in)
	if err != nil {
		return nil, err
	}
	hash, err := utils.HashPassword(in.Password)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}
	a := &model.AdminUser{Username: in.Username, Email: in.Email, PasswordHash: hash, IsActive: true}
	if err := s.repo.Create(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

// UpdateAdminInput changes an admin: activate/deactivate and/or set a new password.
type UpdateAdminInput struct {
	IsActive *bool   `json:"is_active"`
	Password *string `json:"password"`
}

// Update applies the change. Deactivating or resetting a password signs the admin out everywhere.
func (s *AdminService) Update(ctx context.Context, actor, id uuid.UUID, in UpdateAdminInput) (*model.AdminUser, error) {
	if in.IsActive == nil && in.Password == nil {
		return nil, apperr.Invalid("nothing to change")
	}
	target, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, apperr.NotFound("admin not found")
	}
	if in.IsActive != nil && !*in.IsActive && id == actor {
		return nil, apperr.Invalid("you can't deactivate your own account")
	}
	if in.Password != nil {
		if err := checkPassword(*in.Password); err != nil {
			return nil, err
		}
		hash, err := utils.HashPassword(*in.Password)
		if err != nil {
			return nil, fmt.Errorf("hashing password: %w", err)
		}
		if err := s.repo.UpdatePassword(ctx, id, hash); err != nil {
			return nil, err
		}
	}
	if in.IsActive != nil {
		if err := s.repo.SetActive(ctx, id, *in.IsActive); err != nil {
			return nil, err
		}
	}
	if (in.IsActive != nil && !*in.IsActive) || in.Password != nil {
		if err := s.sessions.DeleteAllUserSessions(ctx, id); err != nil {
			return nil, fmt.Errorf("signing the admin out: %w", err)
		}
	}
	return s.repo.GetByID(ctx, id)
}

// Upsert creates the admin, or (if the email exists) sets its password and reactivates it.
// Used by the "admin-upsert" command to set up accounts on a server.
func (s *AdminService) Upsert(ctx context.Context, in CreateAdminInput) (*model.AdminUser, bool, error) {
	in, err := normalizeAdmin(in)
	if err != nil {
		return nil, false, err
	}
	existing, err := s.repo.GetByUsernameOrEmail(ctx, in.Email)
	if err != nil {
		return nil, false, err
	}
	if existing == nil {
		a, err := s.Create(ctx, in)
		return a, true, err
	}
	active, pw := true, in.Password
	a, err := s.Update(ctx, uuid.Nil, existing.ID, UpdateAdminInput{IsActive: &active, Password: &pw})
	return a, false, err
}
