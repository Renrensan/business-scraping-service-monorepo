package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"business-scraping-service/backend/internal/model"
)

type JobRepository struct {
	db *pgxpool.Pool
}

func NewJobRepository(db *pgxpool.Pool) *JobRepository {
	return &JobRepository{db: db}
}

func (r *JobRepository) Create(ctx context.Context, job *model.ScrapeJob) error {
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	now := time.Now().UTC()
	job.CreatedAt = now
	job.UpdatedAt = now

	metaJSON, _ := json.Marshal(job.Metadata)
	if len(metaJSON) == 0 {
		metaJSON = []byte("{}")
	}

	query := `
		INSERT INTO scrape_jobs (
			id, status, query, location, category, max_results, enrich_contacts,
			total_scraped, total_enriched, error_count, error_message, metadata,
			started_at, completed_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
	`
	_, err := r.db.Exec(ctx, query,
		job.ID,
		job.Status,
		job.Query,
		job.Location,
		job.Category,
		job.MaxResults,
		job.EnrichContacts,
		job.TotalScraped,
		job.TotalEnriched,
		job.ErrorCount,
		job.ErrorMessage,
		metaJSON,
		job.StartedAt,
		job.CompletedAt,
		job.CreatedAt,
		job.UpdatedAt,
	)
	return err
}

func (r *JobRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ScrapeJob, error) {
	query := `
		SELECT id, status, query, location, category, max_results, enrich_contacts,
		       total_scraped, total_enriched, error_count, error_message, metadata,
		       started_at, completed_at, created_at, updated_at
		FROM scrape_jobs
		WHERE id = $1
	`
	var job model.ScrapeJob
	var metaBytes []byte
	err := r.db.QueryRow(ctx, query, id).Scan(
		&job.ID,
		&job.Status,
		&job.Query,
		&job.Location,
		&job.Category,
		&job.MaxResults,
		&job.EnrichContacts,
		&job.TotalScraped,
		&job.TotalEnriched,
		&job.ErrorCount,
		&job.ErrorMessage,
		&metaBytes,
		&job.StartedAt,
		&job.CompletedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query job: %w", err)
	}

	if len(metaBytes) > 0 {
		_ = json.Unmarshal(metaBytes, &job.Metadata)
	}
	return &job, nil
}

func (r *JobRepository) List(ctx context.Context, page, pageSize int) ([]*model.ScrapeJob, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var totalCount int64
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM scrape_jobs").Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, status, query, location, category, max_results, enrich_contacts,
		       total_scraped, total_enriched, error_count, error_message, metadata,
		       started_at, completed_at, created_at, updated_at
		FROM scrape_jobs
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []*model.ScrapeJob
	for rows.Next() {
		var job model.ScrapeJob
		var metaBytes []byte
		if err := rows.Scan(
			&job.ID,
			&job.Status,
			&job.Query,
			&job.Location,
			&job.Category,
			&job.MaxResults,
			&job.EnrichContacts,
			&job.TotalScraped,
			&job.TotalEnriched,
			&job.ErrorCount,
			&job.ErrorMessage,
			&metaBytes,
			&job.StartedAt,
			&job.CompletedAt,
			&job.CreatedAt,
			&job.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		if len(metaBytes) > 0 {
			_ = json.Unmarshal(metaBytes, &job.Metadata)
		}
		jobs = append(jobs, &job)
	}

	return jobs, totalCount, nil
}

func (r *JobRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.JobStatus, totalScraped, totalEnriched, errorCount int, errMsg string) error {
	now := time.Now().UTC()
	var startedAtQuery string
	var completedAtQuery string

	if status == model.JobStatusProcessing {
		startedAtQuery = ", started_at = COALESCE(started_at, NOW())"
	}
	if status == model.JobStatusCompleted || status == model.JobStatusFailed || status == model.JobStatusCancelled {
		completedAtQuery = ", completed_at = NOW()"
	}

	query := fmt.Sprintf(`
		UPDATE scrape_jobs
		SET status = $1, total_scraped = $2, total_enriched = $3, error_count = $4, error_message = $5, updated_at = $6 %s %s
		WHERE id = $7
	`, startedAtQuery, completedAtQuery)

	_, err := r.db.Exec(ctx, query, status, totalScraped, totalEnriched, errorCount, errMsg, now, id)
	return err
}

func (r *JobRepository) IncrementProgress(ctx context.Context, id uuid.UUID, scrapedDelta, enrichedDelta, errorDelta int) error {
	query := `
		UPDATE scrape_jobs
		SET total_scraped = total_scraped + $1,
		    total_enriched = total_enriched + $2,
		    error_count = error_count + $3,
		    updated_at = NOW()
		WHERE id = $4
	`
	_, err := r.db.Exec(ctx, query, scrapedDelta, enrichedDelta, errorDelta, id)
	return err
}
