package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SchedulerState is the scheduler's persisted timing and the Places quota pause.
type SchedulerState struct {
	NextRunAt        *time.Time
	LastRunAt        *time.Time
	QuotaPausedUntil *time.Time
}

type SchedulerStateRepository struct {
	db *pgxpool.Pool
}

func NewSchedulerStateRepository(db *pgxpool.Pool) *SchedulerStateRepository {
	return &SchedulerStateRepository{db: db}
}

func (r *SchedulerStateRepository) Get(ctx context.Context) (*SchedulerState, error) {
	var s SchedulerState
	err := r.db.QueryRow(ctx, `SELECT next_run_at, last_run_at, quota_paused_until FROM scheduler_state WHERE id = 1`).
		Scan(&s.NextRunAt, &s.LastRunAt, &s.QuotaPausedUntil)
	if errors.Is(err, pgx.ErrNoRows) {
		return &s, nil
	}
	return &s, err
}

func (r *SchedulerStateRepository) SetNextRun(ctx context.Context, next time.Time) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO scheduler_state (id, next_run_at, updated_at) VALUES (1, $1, NOW())
		ON CONFLICT (id) DO UPDATE SET next_run_at = EXCLUDED.next_run_at, updated_at = NOW()`, next)
	return err
}

// ClaimDueRun atomically moves next_run_at forward if it is due. Only the caller that gets true may
// start the run, so two instances (or a restart racing a tick) never start the same run twice.
func (r *SchedulerStateRepository) ClaimDueRun(ctx context.Context, now time.Time, interval time.Duration) (bool, time.Time, error) {
	next := now.Add(interval)
	tag, err := r.db.Exec(ctx, `
		UPDATE scheduler_state
		SET last_run_at = $1, next_run_at = $2, updated_at = NOW()
		WHERE id = 1 AND next_run_at IS NOT NULL AND next_run_at <= $1`, now, next)
	if err != nil {
		return false, time.Time{}, err
	}
	return tag.RowsAffected() == 1, next, nil
}

// MarkRunNow records a manual run: last run now, next run one interval later.
func (r *SchedulerStateRepository) MarkRunNow(ctx context.Context, now time.Time, interval time.Duration) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO scheduler_state (id, last_run_at, next_run_at, updated_at) VALUES (1, $1, $2, NOW())
		ON CONFLICT (id) DO UPDATE SET last_run_at = EXCLUDED.last_run_at, next_run_at = EXCLUDED.next_run_at, updated_at = NOW()`,
		now, now.Add(interval))
	return err
}

func (r *SchedulerStateRepository) SetQuotaPausedUntil(ctx context.Context, until *time.Time) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO scheduler_state (id, quota_paused_until, updated_at) VALUES (1, $1, NOW())
		ON CONFLICT (id) DO UPDATE SET quota_paused_until = EXCLUDED.quota_paused_until, updated_at = NOW()`, until)
	return err
}
