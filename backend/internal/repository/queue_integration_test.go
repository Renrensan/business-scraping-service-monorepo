package repository

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"business-scraping-service/backend/internal/model"
)

// These tests need a real, disposable Postgres: TEST_DATABASE_URL=postgres://... go test ./internal/repository/
// The database is wiped.
func testDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := InitDB(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	// Migrations must be safe to run on every boot.
	if err := runMigrations(ctx, pool); err != nil {
		t.Fatalf("second migration run failed: %v", err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE business_leads, scrape_jobs, scraper_keywords CASCADE`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM areas WHERE source <> 'preset'`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE places_usage`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE scheduler_state SET next_run_at = NULL, last_run_at = NULL, quota_paused_until = NULL`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestQueue_ConcurrentClaimsNeverShareAJob(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	jobs := NewJobRepository(pool)

	for i := 0; i < 30; i++ {
		if err := jobs.Create(ctx, &model.ScrapeJob{Query: "q", Location: "l"}); err != nil {
			t.Fatal(err)
		}
	}

	var mu sync.Mutex
	seen := map[uuid.UUID]int{}
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				j, err := jobs.ClaimNext(ctx)
				if err != nil {
					t.Error(err)
					return
				}
				if j == nil {
					return
				}
				mu.Lock()
				seen[j.ID]++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if len(seen) != 30 {
		t.Errorf("claimed %d distinct jobs, want 30", len(seen))
	}
	for id, n := range seen {
		if n != 1 {
			t.Errorf("job %s claimed %d times", id, n)
		}
	}
	pending, processing, _ := jobs.QueueStats(ctx)
	if pending != 0 || processing != 30 {
		t.Errorf("pending=%d processing=%d", pending, processing)
	}
}

func TestQueue_RecoverStaleRequeuesThenGivesUp(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	jobs := NewJobRepository(pool)

	fresh := &model.ScrapeJob{Query: "fresh", Location: "l"}
	stale := &model.ScrapeJob{Query: "stale", Location: "l"}
	worn := &model.ScrapeJob{Query: "worn", Location: "l"}
	for _, j := range []*model.ScrapeJob{fresh, stale, worn} {
		if err := jobs.Create(ctx, j); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 3; i++ {
		if _, err := jobs.ClaimNext(ctx); err != nil {
			t.Fatal(err)
		}
	}
	_, _ = pool.Exec(ctx, `UPDATE scrape_jobs SET heartbeat_at = NOW() - INTERVAL '10 minutes' WHERE query IN ('stale','worn')`)
	_, _ = pool.Exec(ctx, `UPDATE scrape_jobs SET attempts = 3 WHERE query = 'worn'`)

	requeued, failed, err := jobs.RecoverStale(ctx, 2*time.Minute, 3)
	if err != nil {
		t.Fatal(err)
	}
	if requeued != 1 || failed != 1 {
		t.Errorf("requeued=%d failed=%d, want 1/1", requeued, failed)
	}
	for q, want := range map[string]model.JobStatus{"fresh": model.JobStatusProcessing, "stale": model.JobStatusPending, "worn": model.JobStatusFailed} {
		var got model.JobStatus
		_ = pool.QueryRow(ctx, `SELECT status FROM scrape_jobs WHERE query = $1`, q).Scan(&got)
		if got != want {
			t.Errorf("%s: status %s, want %s", q, got, want)
		}
	}

	// Requeued job is claimable again, with its attempt counted.
	j, _ := jobs.ClaimNext(ctx)
	if j == nil || j.Query != "stale" || j.Attempts != 2 {
		t.Errorf("reclaimed %+v", j)
	}
}

func TestQueue_CancelPendingSkipsWebAndRunningJobs(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	jobs := NewJobRepository(pool)

	running := &model.ScrapeJob{Query: "running", Location: "l"}
	_ = jobs.Create(ctx, running)
	_, _ = jobs.ClaimNext(ctx)
	_ = jobs.Create(ctx, &model.ScrapeJob{Query: "places", Location: "l"})
	_ = jobs.Create(ctx, &model.ScrapeJob{Query: "web", Location: "l", Metadata: map[string]interface{}{"engine": "web"}})

	n, err := jobs.CancelPendingPlacesJobs(ctx, "quota")
	if err != nil || n != 1 {
		t.Fatalf("cancelled %d, err %v", n, err)
	}
	var webStatus, runStatus string
	_ = pool.QueryRow(ctx, `SELECT status FROM scrape_jobs WHERE query='web'`).Scan(&webStatus)
	_ = pool.QueryRow(ctx, `SELECT status FROM scrape_jobs WHERE query='running'`).Scan(&runStatus)
	if webStatus != "PENDING" || runStatus != "PROCESSING" {
		t.Errorf("web=%s running=%s", webStatus, runStatus)
	}
}

func TestKeywords_RestoreOrphanedProcessing(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	kws := NewKeywordRepository(pool)
	jobs := NewJobRepository(pool)

	orphanReady, _ := kws.Create(ctx, "orphan ready")
	orphanPaging, _ := kws.Create(ctx, "orphan paging")
	busy, _ := kws.Create(ctx, "busy")
	for _, k := range []*model.ScraperKeyword{orphanReady, orphanPaging, busy} {
		_ = kws.SetStatus(ctx, k.ID, model.KeywordStatusProcessing)
	}
	_, _ = pool.Exec(ctx, `UPDATE scraper_keywords SET next_page_token = 'tok' WHERE id = $1`, orphanPaging.ID)
	busyID := busy.ID
	_ = jobs.Create(ctx, &model.ScrapeJob{Query: "busy", Location: "l", Kind: model.JobKindKeyword, KeywordID: &busyID})

	n, err := kws.RestoreOrphanedProcessing(ctx)
	if err != nil || n != 2 {
		t.Fatalf("restored %d, err %v", n, err)
	}
	for id, want := range map[uuid.UUID]model.KeywordStatus{orphanReady.ID: "ready", orphanPaging.ID: "paginating", busy.ID: "processing"} {
		k, _ := kws.GetByID(ctx, id)
		if k.Status != want {
			t.Errorf("%s: %s, want %s", k.Keyword, k.Status, want)
		}
	}
	if active, _ := jobs.HasActiveKeywordJob(ctx, busy.ID); !active {
		t.Error("busy keyword should have an active job")
	}
}

func TestSchedulerState_OnlyOneClaimPerRun(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	st := NewSchedulerStateRepository(pool)

	now := time.Now().UTC().Truncate(time.Second)
	_ = st.SetNextRun(ctx, now.Add(-time.Second))

	var wg sync.WaitGroup
	var mu sync.Mutex
	claims := 0
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, _, err := st.ClaimDueRun(ctx, now, 24*time.Hour)
			if err != nil {
				t.Error(err)
			}
			if ok {
				mu.Lock()
				claims++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if claims != 1 {
		t.Errorf("claims = %d, want exactly 1", claims)
	}
	s, _ := st.Get(ctx)
	if s.NextRunAt == nil || !s.NextRunAt.Equal(now.Add(24*time.Hour)) || s.LastRunAt == nil {
		t.Errorf("state = %+v", s)
	}

	// Not due yet: no claim.
	if ok, _, _ := st.ClaimDueRun(ctx, now.Add(time.Hour), 24*time.Hour); ok {
		t.Error("claimed a run that is not due")
	}

	until := now.Add(5 * time.Hour)
	_ = st.SetQuotaPausedUntil(ctx, &until)
	s, _ = st.Get(ctx)
	if s.QuotaPausedUntil == nil || !s.QuotaPausedUntil.Equal(until) {
		t.Errorf("quota pause not saved: %+v", s.QuotaPausedUntil)
	}
}

func TestLeads_ListPendingEnrichment(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	jobs := NewJobRepository(pool)
	leads := NewBusinessRepository(pool)

	enrichJob := &model.ScrapeJob{Query: "a", Location: "l", EnrichContacts: true}
	noEnrichJob := &model.ScrapeJob{Query: "b", Location: "l", EnrichContacts: false}
	_ = jobs.Create(ctx, enrichJob)
	_ = jobs.Create(ctx, noEnrichJob)

	mk := func(name, site string, job *model.ScrapeJob) *model.BusinessLead {
		l := &model.BusinessLead{Name: name, Address: name + " st", Website: site, JobID: &job.ID, EnrichmentStatus: model.EnrichmentNone}
		if _, err := leads.Upsert(ctx, l); err != nil {
			t.Fatal(err)
		}
		return l
	}
	want := mk("old with site", "https://old.example", enrichJob)
	mk("no site", "", enrichJob)
	mk("enrich off", "https://off.example", noEnrichJob)
	recent := mk("recent", "https://recent.example", enrichJob)
	_, _ = pool.Exec(ctx, `UPDATE business_leads SET last_scraped_at = NOW() - INTERVAL '1 hour' WHERE id <> $1`, recent.ID)

	got, err := leads.ListPendingEnrichment(ctx, 15*time.Minute, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != want.ID || got[0].Website != "https://old.example" {
		t.Errorf("got %d leads: %+v", len(got), got)
	}
}
