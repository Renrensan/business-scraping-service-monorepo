package repository

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"business-scraping-service/backend/internal/model"
)

func TestAreas_PresetsCreateAndDelete(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	areas := NewAreaRepository(pool)
	kws := NewKeywordRepository(pool)

	list, err := areas.List(ctx)
	if err != nil || len(list) < 30 {
		t.Fatalf("expected Bali presets, got %d (%v)", len(list), err)
	}
	canggu, _ := areas.GetByName(ctx, "canggu")
	if canggu == nil || canggu.Source != "preset" || !(canggu.LowLat < canggu.HighLat) {
		t.Fatalf("canggu preset: %+v", canggu)
	}

	a := &model.Area{Name: "Bedugul", LowLat: -8.30, LowLng: 115.15, HighLat: -8.26, HighLng: 115.19}
	if err := areas.Create(ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := areas.Create(ctx, &model.Area{Name: "bedugul", LowLat: -1, LowLng: 1, HighLat: 0, HighLng: 2}); !errors.Is(err, ErrAreaExists) {
		t.Errorf("duplicate name: %v", err)
	}
	if err := areas.Create(ctx, &model.Area{Name: "Backwards", LowLat: 0, LowLng: 0, HighLat: -1, HighLng: 1}); err == nil {
		t.Error("inverted rectangle accepted")
	}

	kw, created, err := kws.CreateAreaKeyword(ctx, " villa ", "lodging", a)
	if err != nil || !created || kw.Keyword != "villa · Bedugul" || kw.Mode != model.KeywordModeArea || kw.AreaName != "Bedugul" || kw.Term != "villa" {
		t.Fatalf("created=%v kw=%+v err=%v", created, kw, err)
	}
	if _, created, _ := kws.CreateAreaKeyword(ctx, "villa", "lodging", a); created {
		t.Error("second create should return the existing keyword")
	}
	if err := areas.Delete(ctx, a.ID); !errors.Is(err, ErrAreaInUse) {
		t.Errorf("deleting an area in use: %v", err)
	}
	_ = kws.Delete(ctx, kw.ID)
	if err := areas.Delete(ctx, a.ID); err != nil {
		t.Errorf("delete unused area: %v", err)
	}
}

func TestCells_SplitAndOrder(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	areas := NewAreaRepository(pool)
	kws := NewKeywordRepository(pool)
	ubud, _ := areas.GetByName(ctx, "Ubud")
	kw, _, _ := kws.CreateAreaKeyword(ctx, "spa", "spa", ubud)

	if err := kws.StartCycle(ctx, kw.ID, ubud); err != nil {
		t.Fatal(err)
	}
	root, _ := kws.NextPendingCell(ctx, kw.ID)
	if root == nil || root.Depth != 0 || root.LowLat != ubud.LowLat {
		t.Fatalf("root %+v", root)
	}
	root.PagesFetched, root.Results = 3, 60
	if err := kws.SplitCell(ctx, root); err != nil {
		t.Fatal(err)
	}
	if n, _ := kws.CountCells(ctx, kw.ID); n != 5 {
		t.Errorf("cells = %d", n)
	}

	// An in-progress cell is resumed before other pending cells.
	first, _ := kws.NextPendingCell(ctx, kw.ID)
	first.PagesFetched, first.NextPageToken = 1, "tok"
	_ = kws.SaveCellProgress(ctx, first)
	again, _ := kws.NextPendingCell(ctx, kw.ID)
	if again.ID != first.ID || again.NextPageToken != "tok" {
		t.Errorf("did not resume in-progress cell")
	}

	got, _ := kws.GetByID(ctx, kw.ID)
	if got.CellsTotal != 4 || got.CellsDone != 0 {
		t.Errorf("progress %d/%d", got.CellsDone, got.CellsTotal)
	}
	// The four quarters exactly cover the parent.
	var area float64
	rows, _ := pool.Query(ctx, `SELECT (high_lat-low_lat)*(high_lng-low_lng) FROM keyword_cells WHERE keyword_id=$1 AND depth=1`, kw.ID)
	for rows.Next() {
		var a float64
		_ = rows.Scan(&a)
		area += a
	}
	rows.Close()
	parent := (ubud.HighLat - ubud.LowLat) * (ubud.HighLng - ubud.LowLng)
	if d := area - parent; d > 1e-9 || d < -1e-9 {
		t.Errorf("quarters cover %f, parent %f", area, parent)
	}

	// Reset clears cells.
	_ = kws.ResetCursor(ctx, kw.ID)
	if n, _ := kws.CountCells(ctx, kw.ID); n != 0 {
		t.Errorf("cells after reset = %d", n)
	}
}

func TestKeywords_BackoffAndPriority(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	kws := NewKeywordRepository(pool)
	now := time.Now().UTC()

	busy, _ := kws.Create(ctx, "busy")
	quiet, _ := kws.Create(ctx, "quiet")
	stale, _ := kws.Create(ctx, "stale")
	for _, k := range []*model.ScraperKeyword{busy, quiet, stale} {
		if _, err := kws.Update(ctx, k.ID, k.Keyword, true); err != nil {
			t.Fatalf("update: %v", err)
		}
	}
	_ = kws.RecordRunOutcome(ctx, busy.ID, 25, now)
	_ = kws.RecordRunOutcome(ctx, quiet.ID, 3, now)
	_ = kws.RecordRunOutcome(ctx, stale.ID, 0, now) // empty run -> back off 1 day
	_ = kws.RecordRunOutcome(ctx, stale.ID, 0, now) // again -> 2 days

	got, _ := kws.GetByID(ctx, stale.ID)
	if got.EmptyRuns != 2 || got.NextEligibleAt == nil || got.NextEligibleAt.Sub(now) < 47*time.Hour || got.NextEligibleAt.Sub(now) > 49*time.Hour {
		t.Errorf("stale backoff: runs=%d next=%v", got.EmptyRuns, got.NextEligibleAt)
	}

	active, _ := kws.GetActiveKeywords(ctx)
	if len(active) != 2 || active[0].Keyword != "busy" || active[1].Keyword != "quiet" {
		names := []string{}
		for _, k := range active {
			names = append(names, k.Keyword)
		}
		t.Errorf("active order = %v, want [busy quiet]", names)
	}

	_ = kws.RecordRunOutcome(ctx, stale.ID, 4, now) // found something: back in rotation
	got, _ = kws.GetByID(ctx, stale.ID)
	if got.EmptyRuns != 0 || got.NextEligibleAt != nil {
		t.Errorf("reset failed: %+v", got)
	}
}

func TestUsage_BudgetIsExactUnderConcurrency(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	usage := NewUsageRepository(pool)
	day := time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC)

	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := usage.TryConsume(ctx, day, 25)
			if err != nil {
				t.Error(err)
			}
			if ok {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	n, _ := usage.Requests(ctx, day)
	if allowed != 25 || n != 25 {
		t.Errorf("allowed=%d counted=%d, want 25/25", allowed, n)
	}
	if ok, _ := usage.TryConsume(ctx, day, 0); !ok {
		t.Error("budget 0 means unlimited")
	}
	if ok, _ := usage.TryConsume(ctx, day.AddDate(0, 0, 1), 25); !ok {
		t.Error("new day should start fresh")
	}
}
