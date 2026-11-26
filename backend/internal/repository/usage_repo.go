package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UsageRepository counts Places API requests per Google quota day (Pacific time).
type UsageRepository struct {
	db *pgxpool.Pool
}

func NewUsageRepository(db *pgxpool.Pool) *UsageRepository {
	return &UsageRepository{db: db}
}

// TryConsume counts one request for day if that keeps the day at or under budget (0 = no budget).
// It returns false, without counting, once the budget is used up. Safe across concurrent workers.
func (r *UsageRepository) TryConsume(ctx context.Context, day time.Time, budget int) (bool, error) {
	if budget <= 0 {
		_, err := r.db.Exec(ctx, `
			INSERT INTO places_usage (day, requests) VALUES ($1, 1)
			ON CONFLICT (day) DO UPDATE SET requests = places_usage.requests + 1`, day)
		return err == nil, err
	}
	var n int
	err := r.db.QueryRow(ctx, `
		INSERT INTO places_usage (day, requests) VALUES ($1, 1)
		ON CONFLICT (day) DO UPDATE SET requests = places_usage.requests + 1
		WHERE places_usage.requests < $2
		RETURNING requests`, day, budget).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (r *UsageRepository) Requests(ctx context.Context, day time.Time) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT requests FROM places_usage WHERE day = $1`, day).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return n, err
}
