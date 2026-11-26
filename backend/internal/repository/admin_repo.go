package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"business-scraping-service/backend/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminRepository struct {
	db *pgxpool.Pool
}

func NewAdminRepository(db *pgxpool.Pool) *AdminRepository {
	return &AdminRepository{db: db}
}

func (r *AdminRepository) CountAll(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM admin_users").Scan(&count)
	return count, err
}

func (r *AdminRepository) Create(ctx context.Context, admin *model.AdminUser) error {
	if admin.ID == uuid.Nil {
		admin.ID = uuid.New()
	}
	now := time.Now().UTC()
	admin.CreatedAt = now
	admin.UpdatedAt = now

	query := `
		INSERT INTO admin_users (id, username, email, password_hash, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(ctx, query,
		admin.ID,
		admin.Username,
		admin.Email,
		admin.PasswordHash,
		admin.IsActive,
		admin.CreatedAt,
		admin.UpdatedAt,
	)
	return err
}

func (r *AdminRepository) GetByUsernameOrEmail(ctx context.Context, identifier string) (*model.AdminUser, error) {
	query := `
		SELECT id, username, email, password_hash, is_active, created_at, updated_at
		FROM admin_users
		WHERE LOWER(username) = LOWER($1) OR LOWER(email) = LOWER($1)
		LIMIT 1
	`
	var admin model.AdminUser
	err := r.db.QueryRow(ctx, query, identifier).Scan(
		&admin.ID,
		&admin.Username,
		&admin.Email,
		&admin.PasswordHash,
		&admin.IsActive,
		&admin.CreatedAt,
		&admin.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query admin: %w", err)
	}
	return &admin, nil
}

func (r *AdminRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.AdminUser, error) {
	query := `
		SELECT id, username, email, password_hash, is_active, created_at, updated_at
		FROM admin_users
		WHERE id = $1
		LIMIT 1
	`
	var admin model.AdminUser
	err := r.db.QueryRow(ctx, query, id).Scan(
		&admin.ID,
		&admin.Username,
		&admin.Email,
		&admin.PasswordHash,
		&admin.IsActive,
		&admin.CreatedAt,
		&admin.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query admin by ID: %w", err)
	}
	return &admin, nil
}

func (r *AdminRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	query := `
		UPDATE admin_users
		SET password_hash = $1, updated_at = $2
		WHERE id = $3
	`
	_, err := r.db.Exec(ctx, query, passwordHash, time.Now().UTC(), id)
	return err
}
