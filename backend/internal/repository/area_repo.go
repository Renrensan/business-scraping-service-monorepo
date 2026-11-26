package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"business-scraping-service/backend/internal/apperr"
	"business-scraping-service/backend/internal/model"
)

var (
	ErrAreaExists = apperr.Conflict("an area with this name already exists")
	ErrAreaInUse  = apperr.Conflict("area is used by keywords; delete them first")
)

type AreaRepository struct {
	db *pgxpool.Pool
}

func NewAreaRepository(db *pgxpool.Pool) *AreaRepository {
	return &AreaRepository{db: db}
}

const areaSelect = `
	SELECT a.id, a.name, a.low_lat, a.low_lng, a.high_lat, a.high_lng, a.source,
	       (SELECT COUNT(*) FROM scraper_keywords k WHERE k.area_id = a.id), a.created_at
	FROM areas a`

func scanArea(row pgx.Row) (*model.Area, error) {
	a := &model.Area{}
	err := row.Scan(&a.ID, &a.Name, &a.LowLat, &a.LowLng, &a.HighLat, &a.HighLng, &a.Source, &a.Keywords, &a.CreatedAt)
	return a, err
}

func (r *AreaRepository) List(ctx context.Context) ([]*model.Area, error) {
	rows, err := r.db.Query(ctx, areaSelect+` ORDER BY a.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Area
	for rows.Next() {
		a, err := scanArea(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *AreaRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Area, error) {
	a, err := scanArea(r.db.QueryRow(ctx, areaSelect+` WHERE a.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

func (r *AreaRepository) GetByName(ctx context.Context, name string) (*model.Area, error) {
	a, err := scanArea(r.db.QueryRow(ctx, areaSelect+` WHERE LOWER(a.name) = LOWER($1)`, strings.TrimSpace(name)))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

func (r *AreaRepository) Create(ctx context.Context, a *model.Area) error {
	a.Name = strings.TrimSpace(a.Name)
	if a.Name == "" {
		return apperr.Invalid("area name is required")
	}
	if !(a.LowLat < a.HighLat && a.LowLng < a.HighLng) {
		return apperr.Invalid("area corners are invalid: south-west must be below and left of north-east")
	}
	if a.LowLat < -90 || a.HighLat > 90 || a.LowLng < -180 || a.HighLng > 180 {
		return apperr.Invalid("area coordinates are out of range")
	}
	if a.Source == "" {
		a.Source = "manual"
	}
	err := r.db.QueryRow(ctx, `
		INSERT INTO areas (name, low_lat, low_lng, high_lat, high_lng, source)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`, a.Name, a.LowLat, a.LowLng, a.HighLat, a.HighLng, a.Source).Scan(&a.ID, &a.CreatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrAreaExists
	}
	return err
}

func (r *AreaRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM areas WHERE id = $1`, id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return ErrAreaInUse
	}
	return err
}
