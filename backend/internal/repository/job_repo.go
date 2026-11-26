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

// scrape_jobs is also the work queue: a job is created PENDING, a worker claims it (PROCESSING) and
// keeps heartbeat_at fresh while it runs, and it ends COMPLETED, FAILED or CANCELLED. A PROCESSING job
// whose heartbeat went stale belongs to a process that died, and is put back in the queue.

type JobRepository struct {
	db *pgxpool.Pool
}

func NewJobRepository(db *pgxpool.Pool) *JobRepository {
	return &JobRepository{db: db}
}

const jobColumns = `id, status, query, location, category, max_results, enrich_contacts,
	total_scraped, total_enriched, error_count, error_message, metadata,
	started_at, completed_at, created_at, updated_at, kind, keyword_id, attempts`

func scanJob(row pgx.Row) (*model.ScrapeJob, error) {
	var job model.ScrapeJob
	var metaBytes []byte
	var errMsg *string
	if err := row.Scan(
		&job.ID, &job.Status, &job.Query, &job.Location, &job.Category, &job.MaxResults, &job.EnrichContacts,
		&job.TotalScraped, &job.TotalEnriched, &job.ErrorCount, &errMsg, &metaBytes,
		&job.StartedAt, &job.CompletedAt, &job.CreatedAt, &job.UpdatedAt, &job.Kind, &job.KeywordID, &job.Attempts,
	); err != nil {
		return nil, err
	}
	if errMsg != nil {
		job.ErrorMessage = *errMsg
	}
	if len(metaBytes) > 0 {
		decodeForDisplay(metaBytes, &job.Metadata, "scrape_jobs", job.ID, "metadata")
	}
	return &job, nil
}

func (r *JobRepository) Create(ctx context.Context, job *model.ScrapeJob) error {
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	if job.Kind == "" {
		job.Kind = model.JobKindManual
	}
	if job.Status == "" {
		job.Status = model.JobStatusPending
	}
	now := time.Now().UTC()
	job.CreatedAt = now
	job.UpdatedAt = now

	metaJSON, _ := json.Marshal(job.Metadata)
	if len(metaJSON) == 0 || string(metaJSON) == "null" {
		metaJSON = []byte("{}")
	}

	query := `
		INSERT INTO scrape_jobs (
			id, status, query, location, category, max_results, enrich_contacts,
			total_scraped, total_enriched, error_count, error_message, metadata,
			started_at, completed_at, created_at, updated_at, kind, keyword_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		)
	`
	_, err := r.db.Exec(ctx, query,
		job.ID, job.Status, job.Query, job.Location, job.Category, job.MaxResults, job.EnrichContacts,
		job.TotalScraped, job.TotalEnriched, job.ErrorCount, job.ErrorMessage, metaJSON,
		job.StartedAt, job.CompletedAt, job.CreatedAt, job.UpdatedAt, job.Kind, job.KeywordID,
	)
	return err
}

func (r *JobRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ScrapeJob, error) {
	job, err := scanJob(r.db.QueryRow(ctx, `SELECT `+jobColumns+` FROM scrape_jobs WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query job: %w", err)
	}
	return job, nil
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
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM scrape_jobs").Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `SELECT `+jobColumns+` FROM scrape_jobs ORDER BY created_at DESC LIMIT $1 OFFSET $2`, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []*model.ScrapeJob
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, job)
	}
	return jobs, totalCount, rows.Err()
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

// ---------------------------------------------------------------------------------------------
// Queue operations

// ClaimNext takes the oldest PENDING job for this worker, or returns nil when the queue is empty.
// SKIP LOCKED lets several workers (or instances) claim concurrently without taking the same job.
func (r *JobRepository) ClaimNext(ctx context.Context) (*model.ScrapeJob, error) {
	query := `
		UPDATE scrape_jobs
		SET status = 'PROCESSING',
		    attempts = attempts + 1,
		    heartbeat_at = NOW(),
		    started_at = COALESCE(started_at, NOW()),
		    updated_at = NOW()
		WHERE id = (
			SELECT id FROM scrape_jobs
			WHERE status = 'PENDING'
			ORDER BY created_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING ` + jobColumns
	job, err := scanJob(r.db.QueryRow(ctx, query))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return job, err
}

// Heartbeat marks a running job as alive.
func (r *JobRepository) Heartbeat(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE scrape_jobs SET heartbeat_at = NOW() WHERE id = $1 AND status = 'PROCESSING'`, id)
	return err
}

// Requeue puts a job that was interrupted (shutdown) back in the queue.
func (r *JobRepository) Requeue(ctx context.Context, id uuid.UUID, msg string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE scrape_jobs
		SET status = 'PENDING', heartbeat_at = NULL, completed_at = NULL, error_message = $2, updated_at = NOW()
		WHERE id = $1`, id, msg)
	return err
}

// RecoverStale requeues PROCESSING jobs whose heartbeat is older than staleAfter (their process died),
// or fails them once they have used maxAttempts. Jobs from before the queue existed have no heartbeat
// and are judged by updated_at.
func (r *JobRepository) RecoverStale(ctx context.Context, staleAfter time.Duration, maxAttempts int) (requeued, failed int, err error) {
	cutoff := time.Now().UTC().Add(-staleAfter)

	tag, err := r.db.Exec(ctx, `
		UPDATE scrape_jobs
		SET status = 'FAILED', completed_at = NOW(), updated_at = NOW(),
		    error_message = 'Interrupted by a restart too many times; giving up'
		WHERE status = 'PROCESSING' AND COALESCE(heartbeat_at, updated_at) < $1 AND attempts >= $2`, cutoff, maxAttempts)
	if err != nil {
		return 0, 0, err
	}
	failed = int(tag.RowsAffected())

	tag, err = r.db.Exec(ctx, `
		UPDATE scrape_jobs
		SET status = 'PENDING', heartbeat_at = NULL, updated_at = NOW(),
		    error_message = 'Interrupted by a restart; queued again'
		WHERE status = 'PROCESSING' AND COALESCE(heartbeat_at, updated_at) < $1 AND attempts < $2`, cutoff, maxAttempts)
	if err != nil {
		return 0, failed, err
	}
	return int(tag.RowsAffected()), failed, nil
}

// CancelPendingPlacesJobs cancels queued jobs that would call the Places API (used when the quota is
// exhausted, so they don't each fail on their own). Web-engine jobs are left alone.
func (r *JobRepository) CancelPendingPlacesJobs(ctx context.Context, msg string) (int, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE scrape_jobs
		SET status = 'CANCELLED', completed_at = NOW(), updated_at = NOW(), error_message = $1
		WHERE status = 'PENDING' AND COALESCE(metadata->>'engine', '') <> 'web'`, msg)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// HasActiveKeywordJob reports whether a keyword already has a queued or running job.
func (r *JobRepository) HasActiveKeywordJob(ctx context.Context, keywordID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM scrape_jobs WHERE keyword_id = $1 AND status IN ('PENDING', 'PROCESSING'))`, keywordID).Scan(&exists)
	return exists, err
}

// QueueStats counts queued and running jobs.
func (r *JobRepository) QueueStats(ctx context.Context) (pending, processing int, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE status = 'PENDING'), COUNT(*) FILTER (WHERE status = 'PROCESSING')
		FROM scrape_jobs WHERE status IN ('PENDING', 'PROCESSING')`).Scan(&pending, &processing)
	return pending, processing, err
}

// JobHealth summarises recent job outcomes for the health panel.
type JobHealth struct {
	Pending        int             `json:"pending"`
	Processing     int             `json:"processing"`
	Completed24h   int             `json:"completed_24h"`
	Failed24h      int             `json:"failed_24h"`
	Cancelled24h   int             `json:"cancelled_24h"`
	NewLeads24h    int             `json:"new_leads_24h"`
	RecentFailures []RecentFailure `json:"recent_failures"`
}

type RecentFailure struct {
	Query        string     `json:"query"`
	Status       string     `json:"status"`
	ErrorMessage string     `json:"error_message"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
}

func (r *JobRepository) Health(ctx context.Context) (*JobHealth, error) {
	h := &JobHealth{RecentFailures: []RecentFailure{}}
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE status = 'PENDING'),
		       COUNT(*) FILTER (WHERE status = 'PROCESSING'),
		       COUNT(*) FILTER (WHERE status = 'COMPLETED' AND completed_at > NOW() - INTERVAL '24 hours'),
		       COUNT(*) FILTER (WHERE status = 'FAILED' AND completed_at > NOW() - INTERVAL '24 hours'),
		       COUNT(*) FILTER (WHERE status = 'CANCELLED' AND completed_at > NOW() - INTERVAL '24 hours'),
		       (SELECT COUNT(*) FROM business_leads WHERE created_at > NOW() - INTERVAL '24 hours')
		FROM scrape_jobs`).Scan(&h.Pending, &h.Processing, &h.Completed24h, &h.Failed24h, &h.Cancelled24h, &h.NewLeads24h)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT query, status, COALESCE(error_message, ''), completed_at
		FROM scrape_jobs
		WHERE status IN ('FAILED', 'CANCELLED') OR (status = 'COMPLETED' AND COALESCE(error_message, '') <> '')
		ORDER BY COALESCE(completed_at, updated_at) DESC
		LIMIT 5`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var f RecentFailure
		if err := rows.Scan(&f.Query, &f.Status, &f.ErrorMessage, &f.FinishedAt); err != nil {
			return nil, err
		}
		h.RecentFailures = append(h.RecentFailures, f)
	}
	return h, rows.Err()
}
