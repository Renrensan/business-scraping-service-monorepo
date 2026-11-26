-- Migration 007: durable job queue and persisted scheduler state.
-- Durable job queue: scrape_jobs rows are claimed by workers and survive restarts.
ALTER TABLE scrape_jobs ADD COLUMN IF NOT EXISTS kind VARCHAR(20) NOT NULL DEFAULT 'manual';
ALTER TABLE scrape_jobs ADD COLUMN IF NOT EXISTS keyword_id UUID;
ALTER TABLE scrape_jobs ADD COLUMN IF NOT EXISTS attempts INT NOT NULL DEFAULT 0;
ALTER TABLE scrape_jobs ADD COLUMN IF NOT EXISTS heartbeat_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_scrape_jobs_pending ON scrape_jobs(created_at) WHERE status = 'PENDING';
CREATE INDEX IF NOT EXISTS idx_scrape_jobs_keyword_active ON scrape_jobs(keyword_id) WHERE status IN ('PENDING', 'PROCESSING');

-- Older keyword jobs were stored as plain jobs: mark them as keyword jobs so a requeue runs the right pipeline.
UPDATE scrape_jobs SET kind = 'keyword', keyword_id = (metadata->>'keyword_id')::uuid
WHERE kind = 'manual' AND metadata->>'source' = 'auto_scraper_keyword' AND metadata->>'keyword_id' ~ '^[0-9a-fA-F-]{36}$';
UPDATE scrape_jobs j SET kind = 'keyword', keyword_id = k.id
FROM scraper_keywords k
WHERE j.kind = 'manual' AND j.category = 'Keyword Scrape' AND j.query = k.keyword;
-- Keyword jobs whose keyword was deleted keep keyword_id NULL, and a queued one fails with "Keyword no longer exists".
UPDATE scrape_jobs SET kind = 'keyword'
WHERE kind = 'manual' AND (category = 'Keyword Scrape' OR metadata->>'source' = 'auto_scraper_keyword');

-- Scheduler state lives in the database so restarts don't move or skip runs, and only one instance
-- can start a given run.
CREATE TABLE IF NOT EXISTS scheduler_state (
    id INT PRIMARY KEY CHECK (id = 1),
    next_run_at TIMESTAMPTZ,
    last_run_at TIMESTAMPTZ,
    quota_paused_until TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
INSERT INTO scheduler_state (id) VALUES (1) ON CONFLICT (id) DO NOTHING;
