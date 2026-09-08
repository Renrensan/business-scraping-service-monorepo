package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func InitDB(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Auto-run migrations if migration files exist
	if err := runMigrations(ctx, pool); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return pool, nil
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrationDirs := []string{"./migrations", "../migrations", "backend/migrations"}
	var migrationFile string

	for _, dir := range migrationDirs {
		candidate := filepath.Join(dir, "001_init_schema.sql")
		if _, err := os.Stat(candidate); err == nil {
			migrationFile = candidate
			break
		}
	}

	if migrationFile == "" {
		return nil // No migration file found, schema might already exist
	}

	content, err := os.ReadFile(migrationFile)
	if err != nil {
		return err
	}

	queries := strings.Split(string(content), ";")
	for _, query := range queries {
		q := strings.TrimSpace(query)
		if q == "" {
			continue
		}
		if _, err := pool.Exec(ctx, q); err != nil {
			return fmt.Errorf("migration query error: %w", err)
		}
	}

	return nil
}
