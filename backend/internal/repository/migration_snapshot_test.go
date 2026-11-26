package repository

import (
	"context"
	"os"
	"testing"
)

// Runs the boot migrations against a restored copy of production data, twice, and checks that legacy
// keyword jobs were linked to their keywords. MIGRATION_SNAPSHOT_URL must point at a disposable copy.
func TestMigrations_OnProductionSnapshot(t *testing.T) {
	url := os.Getenv("MIGRATION_SNAPSHOT_URL")
	if url == "" {
		t.Skip("MIGRATION_SNAPSHOT_URL not set")
	}
	ctx := context.Background()
	pool, err := InitDB(ctx, url) // first run
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := runMigrations(ctx, pool); err != nil { // second run must be a no-op
		t.Fatal(err)
	}

	var legacy, linked, unlinked int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE category = 'Keyword Scrape' OR metadata->>'source' = 'auto_scraper_keyword'),
		       COUNT(*) FILTER (WHERE kind = 'keyword' AND keyword_id IS NOT NULL),
		       -- the rest belong to deleted keywords
		       COUNT(*) FILTER (WHERE (category = 'Keyword Scrape' OR metadata->>'source' = 'auto_scraper_keyword') AND kind <> 'keyword')
		FROM scrape_jobs`).Scan(&legacy, &linked, &unlinked)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("legacy keyword jobs=%d linked=%d unlinked=%d", legacy, linked, unlinked)
	if unlinked != 0 {
		t.Errorf("%d legacy keyword jobs not linked to a keyword", unlinked)
	}

	var orphanActive int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM scrape_jobs WHERE kind = 'keyword' AND keyword_id IS NULL AND status IN ('PENDING', 'PROCESSING')`).Scan(&orphanActive)
	if orphanActive != 0 {
		t.Errorf("%d active keyword jobs have no keyword", orphanActive)
	}

	var stateRows int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM scheduler_state`).Scan(&stateRows)
	if stateRows != 1 {
		t.Errorf("scheduler_state rows = %d", stateRows)
	}
}
