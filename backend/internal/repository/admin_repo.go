package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"business-scraping-service/backend/internal/apperr"
	"business-scraping-service/backend/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrAdminExists
	}
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

// ErrAdminExists is returned when the username or email is already taken.
var ErrAdminExists = apperr.Conflict("an admin with this username or email already exists")

// List returns all admins (without password hashes), active first.
func (r *AdminRepository) List(ctx context.Context) ([]model.AdminUser, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, username, email, is_active, created_at, updated_at, last_login_at
		FROM admin_users
		ORDER BY is_active DESC, created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.AdminUser{}
	for rows.Next() {
		var a model.AdminUser
		if err := rows.Scan(&a.ID, &a.Username, &a.Email, &a.IsActive, &a.CreatedAt, &a.UpdatedAt, &a.LastLoginAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// SetActive activates or deactivates an admin. Deactivating the last active admin is refused
// in the same statement, so two concurrent requests can't lock everyone out.
func (r *AdminRepository) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE admin_users SET is_active = $2, updated_at = NOW()
		WHERE id = $1
		  AND ($2 OR (SELECT COUNT(*) FROM admin_users WHERE is_active AND id <> $1) > 0)`, id, active)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		if a, err := r.GetByID(ctx, id); err == nil && a == nil {
			return apperr.NotFound("admin not found")
		}
		return apperr.Conflict("can't deactivate the last active admin")
	}
	return nil
}

// TouchLastLogin records a successful login.
func (r *AdminRepository) TouchLastLogin(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE admin_users SET last_login_at = NOW() WHERE id = $1`, id)
	return err
}
