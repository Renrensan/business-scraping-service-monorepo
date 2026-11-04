package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SettingsRepository struct {
	db *pgxpool.Pool
}

func NewSettingsRepository(db *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{db: db}
}

func (r *SettingsRepository) Get(ctx context.Context, key string, target interface{}) error {
	query := `SELECT value FROM system_settings WHERE key = $1 LIMIT 1`
	var valBytes []byte
	err := r.db.QueryRow(ctx, query, key).Scan(&valBytes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	return json.Unmarshal(valBytes, target)
}

func (r *SettingsRepository) Set(ctx context.Context, key string, value interface{}) error {
	valBytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	query := `
		INSERT INTO system_settings (key, value, updated_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE SET
			value = EXCLUDED.value,
			updated_at = EXCLUDED.updated_at
	`
	_, err = r.db.Exec(ctx, query, key, valBytes, now)
	return err
}
