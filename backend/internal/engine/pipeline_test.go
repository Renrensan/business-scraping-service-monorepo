package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"business-scraping-service/backend/internal/engine/enrichment"
	"business-scraping-service/backend/internal/engine/gmaps"
	"business-scraping-service/backend/internal/model"
)

// ---- fakes ----

type fakePlaces struct {
	calls   []string // page tokens requested
	specs   []gmaps.SearchSpec
	respond func(token string) ([]*model.BusinessLead, string, int, bool, error)
}

func (f *fakePlaces) SearchPlacesText(ctx context.Context, q string, max int) ([]*model.BusinessLead, error) {
	return nil, errors.New("not used")
}

func (f *fakePlaces) SearchWithCursor(ctx context.Context, spec gmaps.SearchSpec, token string, maxPages, pageSize int) ([]*model.BusinessLead, string, int, bool, error) {
	f.calls = append(f.calls, token)
	f.specs = append(f.specs, spec)
	return f.respond(token)
}

type fakeLeads struct {
	mu        sync.Mutex
	failAll   bool
	saved     []*model.BusinessLead
	enrichSet map[uuid.UUID]model.EnrichmentStatus
}

func (f *fakeLeads) Upsert(ctx context.Context, l *model.BusinessLead) (bool, error) {
	if f.failAll {
		return false, errors.New("db down")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	l.ID = uuid.New()
	f.saved = append(f.saved, l)
	return true, nil
}

func (f *fakeLeads) UpdateEnrichment(ctx context.Context, id uuid.UUID, e, p []string, fb, ig, li, tw, yt, tt string, s model.EnrichmentStatus) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.enrichSet == nil {
		f.enrichSet = map[uuid.UUID]model.EnrichmentStatus{}
	}
	f.enrichSet[id] = s
	return nil
}

type jobUpdate struct {
	status  model.JobStatus
	scraped int
	errs    int
	msg     string
}

type fakeJobs struct {
	mu      sync.Mutex
	updates []jobUpdate
}

func (f *fakeJobs) UpdateStatus(ctx context.Context, id uuid.UUID, s model.JobStatus, scraped, enriched, errs int, msg string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updates = append(f.updates, jobUpdate{s, scraped, errs, msg})
	return nil
}
func (f *fakeJobs) IncrementProgress(ctx context.Context, id uuid.UUID, a, b, c int) error {
	return nil
}
func (f *fakeJobs) last() jobUpdate {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.updates[len(f.updates)-1]
}

type cursorUpdate struct {
	token  string
	page   int
	status model.KeywordStatus
}

type fakeKeywords struct {
	cursor    *cursorUpdate
	status    model.KeywordStatus
	resets    int
	cursorErr error
	outcomes  []int
	cells     []*model.KeywordCell
}

func (f *fakeKeywords) RecordRunOutcome(ctx context.Context, id uuid.UUID, n int, now time.Time) error {
	f.outcomes = append(f.outcomes, n)
	return nil
}
func (f *fakeKeywords) CountCells(ctx context.Context, id uuid.UUID) (int, error) { return len(f.cells), nil }
func (f *fakeKeywords) StartCycle(ctx context.Context, id uuid.UUID, a *model.Area) error {
	f.cells = []*model.KeywordCell{{ID: uuid.New(), KeywordID: id, Status: "pending", LowLat: a.LowLat, LowLng: a.LowLng, HighLat: a.HighLat, HighLng: a.HighLng}}
	return nil
}
func (f *fakeKeywords) NextPendingCell(ctx context.Context, id uuid.UUID) (*model.KeywordCell, error) {
	var best *model.KeywordCell
	for _, c := range f.cells {
		if c.Status != "pending" {
			continue
		}
		if best == nil || (c.PagesFetched > 0 && best.PagesFetched == 0) || (c.PagesFetched > 0) == (best.PagesFetched > 0) && c.Depth < best.Depth {
			best = c
		}
	}
	if best == nil {
		return nil, nil
	}
	cp := *best
	return &cp, nil
}
func (f *fakeKeywords) find(id uuid.UUID) *model.KeywordCell {
	for _, c := range f.cells {
		if c.ID == id {
			return c
		}
	}
	return nil
}
func (f *fakeKeywords) SaveCellProgress(ctx context.Context, c *model.KeywordCell) error {
	*f.find(c.ID) = *c
	return nil
}
func (f *fakeKeywords) SplitCell(ctx context.Context, c *model.KeywordCell) error {
	stored := f.find(c.ID)
	*stored = *c
	stored.Status = "split"
	midLat, midLng := (c.LowLat+c.HighLat)/2, (c.LowLng+c.HighLng)/2
	for _, q := range [4][4]float64{{c.LowLat, c.LowLng, midLat, midLng}, {c.LowLat, midLng, midLat, c.HighLng}, {midLat, c.LowLng, c.HighLat, midLng}, {midLat, midLng, c.HighLat, c.HighLng}} {
		f.cells = append(f.cells, &model.KeywordCell{ID: uuid.New(), KeywordID: c.KeywordID, Depth: c.Depth + 1, Status: "pending", LowLat: q[0], LowLng: q[1], HighLat: q[2], HighLng: q[3]})
	}
	return nil
}
func (f *fakeKeywords) CycleResults(ctx context.Context, id uuid.UUID) (int, error) {
	n := 0
	for _, c := range f.cells {
		if c.Status != "split" {
			n += c.Results
		}
	}
	return n, nil
}

type fakeAreas struct{ area *model.Area }

func (f fakeAreas) GetByID(ctx context.Context, id uuid.UUID) (*model.Area, error) { return f.area, nil }

func (f *fakeKeywords) UpdateCursor(ctx context.Context, id uuid.UUID, tok string, page, add int, s model.KeywordStatus) error {
	if f.cursorErr != nil {
		return f.cursorErr
	}
	f.cursor = &cursorUpdate{tok, page, s}
	f.status = s
	return nil
}
func (f *fakeKeywords) SetStatus(ctx context.Context, id uuid.UUID, s model.KeywordStatus) error {
	f.status = s
	return nil
}
func (f *fakeKeywords) ResetCursor(ctx context.Context, id uuid.UUID) error {
	f.resets++
	return nil
}

func leadsN(n int) []*model.BusinessLead {
	out := make([]*model.BusinessLead, n)
	for i := range out {
		out[i] = &model.BusinessLead{Name: fmt.Sprintf("Place %d", i), Category: "Hotel"}
	}
	return out
}

func newTestRunner(places *fakePlaces, kws *fakeKeywords, leads *fakeLeads, jobs *fakeJobs) *KeywordRunner {
	r := NewKeywordRunner(kws, fakeAreas{testArea}, leads, jobs, nil, func() string { return "key" })
	r.SetPlacesFactory(func(string) PlacesSearcher { return places })
	return r
}

// ---- KeywordRunner ----

func TestKeywordRunner_FirstRunPaginates(t *testing.T) {
	places := &fakePlaces{respond: func(string) ([]*model.BusinessLead, string, int, bool, error) {
		return leadsN(40), "tok3", 2, false, nil
	}}
	kws, leads, jobs := &fakeKeywords{}, &fakeLeads{}, &fakeJobs{}
	kw := &model.ScraperKeyword{ID: uuid.New(), Keyword: "Hotel in Kuta, Bali", Status: model.KeywordStatusReady, CurrentPage: 1}

	res, err := newTestRunner(places, kws, leads, jobs).Run(context.Background(), kw, uuid.New(), KeywordRunOptions{MaxPages: 2})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != model.KeywordStatusPaginating || kws.cursor.token != "tok3" || kws.cursor.page != 3 {
		t.Errorf("status=%s cursor=%+v", res.Status, kws.cursor)
	}
	if got := jobs.last(); got.status != model.JobStatusCompleted || got.scraped != 40 {
		t.Errorf("job = %+v", got)
	}
	if len(leads.saved) != 40 || leads.saved[0].ScrapeKeywords[0] != kw.Keyword || leads.saved[0].Category != "Hotel" {
		t.Errorf("leads not tagged correctly: %+v", leads.saved[0])
	}
}

func TestKeywordRunner_ContinuesFromSavedCursor(t *testing.T) {
	places := &fakePlaces{respond: func(string) ([]*model.BusinessLead, string, int, bool, error) {
		return leadsN(20), "", 1, true, nil
	}}
	kws := &fakeKeywords{}
	kw := &model.ScraperKeyword{ID: uuid.New(), Keyword: "x", Status: model.KeywordStatusPaginating, NextPageToken: "tok3", CurrentPage: 3}

	res, _ := newTestRunner(places, kws, &fakeLeads{}, &fakeJobs{}).Run(context.Background(), kw, uuid.New(), KeywordRunOptions{MaxPages: 3})
	if places.calls[0] != "tok3" {
		t.Errorf("expected run to continue from tok3, got %q", places.calls[0])
	}
	if res.Status != model.KeywordStatusCompleted || kws.cursor.page != 4 || kws.cursor.token != "" {
		t.Errorf("status=%s cursor=%+v", res.Status, kws.cursor)
	}
}

func TestKeywordRunner_EmptyFirstPageIsNoResults(t *testing.T) {
	places := &fakePlaces{respond: func(string) ([]*model.BusinessLead, string, int, bool, error) {
		return nil, "", 1, true, nil
	}}
	kws, jobs := &fakeKeywords{}, &fakeJobs{}
	kw := &model.ScraperKeyword{ID: uuid.New(), Keyword: "Villa in Jimbaran, Bali", Status: model.KeywordStatusCompleted, CurrentPage: 2}

	res, err := newTestRunner(places, kws, &fakeLeads{}, jobs).Run(context.Background(), kw, uuid.New(), KeywordRunOptions{MaxPages: 3})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != model.KeywordStatusNoResults || jobs.last().msg == "" {
		t.Errorf("status=%s job=%+v", res.Status, jobs.last())
	}
}

func TestKeywordRunner_QuotaErrorKeepsCursor(t *testing.T) {
	places := &fakePlaces{respond: func(string) ([]*model.BusinessLead, string, int, bool, error) {
		return nil, "", 0, false, fmt.Errorf("%w: [RESOURCE_EXHAUSTED] quota", gmaps.ErrQuotaExceeded)
	}}
	kws, jobs := &fakeKeywords{}, &fakeJobs{}
	kw := &model.ScraperKeyword{ID: uuid.New(), Keyword: "x", Status: model.KeywordStatusPaginating, NextPageToken: "tok2", CurrentPage: 2}

	_, err := newTestRunner(places, kws, &fakeLeads{}, jobs).Run(context.Background(), kw, uuid.New(), KeywordRunOptions{MaxPages: 3})
	if !gmaps.IsQuotaError(err) {
		t.Fatalf("expected quota error, got %v", err)
	}
	if kws.cursor != nil || kws.resets != 0 {
		t.Errorf("cursor must be untouched on quota error: cursor=%+v resets=%d", kws.cursor, kws.resets)
	}
	if kws.status != model.KeywordStatusPaginating {
		t.Errorf("keyword status should be restored to paginating, got %s", kws.status)
	}
	if jobs.last().status != model.JobStatusFailed {
		t.Errorf("job = %+v", jobs.last())
	}
}

func TestKeywordRunner_ExpiredTokenRestartsFromPageOne(t *testing.T) {
	places := &fakePlaces{respond: func(tok string) ([]*model.BusinessLead, string, int, bool, error) {
		if tok != "" {
			return nil, "", 0, false, fmt.Errorf("%w: expired", gmaps.ErrInvalidPageToken)
		}
		return leadsN(20), "tok2", 1, false, nil
	}}
	kws := &fakeKeywords{}
	kw := &model.ScraperKeyword{ID: uuid.New(), Keyword: "x", Status: model.KeywordStatusPaginating, NextPageToken: "old", CurrentPage: 3}

	res, err := newTestRunner(places, kws, &fakeLeads{}, &fakeJobs{}).Run(context.Background(), kw, uuid.New(), KeywordRunOptions{MaxPages: 1})
	if err != nil {
		t.Fatal(err)
	}
	if kws.resets != 1 || len(places.calls) != 2 || res.StartPage != 1 || kws.cursor.page != 2 {
		t.Errorf("resets=%d calls=%v start=%d cursor=%+v", kws.resets, places.calls, res.StartPage, kws.cursor)
	}
}

func TestKeywordRunner_NothingSavedKeepsCursorAndFailsJob(t *testing.T) {
	places := &fakePlaces{respond: func(string) ([]*model.BusinessLead, string, int, bool, error) {
		return leadsN(20), "tok2", 1, false, nil
	}}
	kws, jobs := &fakeKeywords{}, &fakeJobs{}
	kw := &model.ScraperKeyword{ID: uuid.New(), Keyword: "x", Status: model.KeywordStatusReady}

	_, err := newTestRunner(places, kws, &fakeLeads{failAll: true}, jobs).Run(context.Background(), kw, uuid.New(), KeywordRunOptions{MaxPages: 1})
	if err == nil || kws.cursor != nil || jobs.last().status != model.JobStatusFailed {
		t.Errorf("err=%v cursor=%+v job=%+v", err, kws.cursor, jobs.last())
	}
}

func TestKeywordRunner_CursorWriteFailureFailsJob(t *testing.T) {
	places := &fakePlaces{respond: func(string) ([]*model.BusinessLead, string, int, bool, error) {
		return leadsN(5), "", 1, true, nil
	}}
	kws, jobs := &fakeKeywords{cursorErr: errors.New("db down")}, &fakeJobs{}
	kw := &model.ScraperKeyword{ID: uuid.New(), Keyword: "x"}

	_, err := newTestRunner(places, kws, &fakeLeads{}, jobs).Run(context.Background(), kw, uuid.New(), KeywordRunOptions{})
	if err == nil || jobs.last().status != model.JobStatusFailed {
		t.Errorf("err=%v job=%+v", err, jobs.last())
	}
}

func TestKeywordRunner_NoAPIKey(t *testing.T) {
	kws, jobs := &fakeKeywords{}, &fakeJobs{}
	r := NewKeywordRunner(kws, nil, &fakeLeads{}, jobs, nil, func() string { return "" })
	_, err := r.Run(context.Background(), &model.ScraperKeyword{ID: uuid.New(), Keyword: "x"}, uuid.New(), KeywordRunOptions{})
	if !errors.Is(err, ErrNoAPIKey) || jobs.last().status != model.JobStatusFailed || kws.status != model.KeywordStatusReady {
		t.Errorf("err=%v job=%+v kwStatus=%s", err, jobs.last(), kws.status)
	}
}

// ---- manual jobs ----

func TestManualJobOutcome(t *testing.T) {
	quota := fmt.Errorf("%w: x", gmaps.ErrQuotaExceeded)
	cases := []struct {
		name   string
		found  int
		ingest IngestResult
		err    error
		want   model.JobStatus
	}{
		{"ok", 10, IngestResult{Saved: 10}, nil, model.JobStatusCompleted},
		{"quota before anything", 0, IngestResult{}, quota, model.JobStatusFailed},
		{"quota after some pages", 20, IngestResult{Saved: 20}, quota, model.JobStatusCompleted},
		{"all filtered out", 20, IngestResult{}, nil, model.JobStatusFailed},
		{"nothing found", 0, IngestResult{}, nil, model.JobStatusFailed},
		{"all saves failed", 5, IngestResult{Failed: 5, LastErr: errors.New("db")}, nil, model.JobStatusFailed},
	}
	for _, c := range cases {
		got, msg := manualJobOutcome(c.found, c.ingest, c.err)
		if got != c.want {
			t.Errorf("%s: got %s (%s), want %s", c.name, got, msg, c.want)
		}
	}
}

// ---- enrichment ----

type fakeCrawler struct{}

func (fakeCrawler) EnrichWebsite(ctx context.Context, site string) (*enrichment.EnrichmentResult, error) {
	if site == "https://broken.example" {
		return nil, errors.New("timeout")
	}
	return &enrichment.EnrichmentResult{Emails: []string{"a@b.c"}}, nil
}

func TestLeadEnricher_SkipsLeadsWithoutWebsiteOrID(t *testing.T) {
	leads := &fakeLeads{}
	e := NewLeadEnricher(fakeCrawler{}, leads, &fakeJobs{}, 2)
	in := []*model.BusinessLead{
		{ID: uuid.New(), Website: "https://ok.example"},
		{ID: uuid.New(), Website: "https://broken.example"},
		{ID: uuid.New(), Website: ""},
		{Website: "https://no-id.example"},
		{ID: uuid.New(), Website: "https://done.example", EnrichmentStatus: model.EnrichmentSuccess},
	}
	if got := e.Enrich(context.Background(), uuid.New(), in); got != 1 {
		t.Errorf("enriched = %d, want 1", got)
	}
	if len(leads.enrichSet) != 2 || leads.enrichSet[in[1].ID] != model.EnrichmentFailed {
		t.Errorf("enrichment writes = %v", leads.enrichSet)
	}
}

// ---- area keywords ----

var testArea = &model.Area{ID: uuid.New(), Name: "Canggu", LowLat: -8.67, LowLng: 115.11, HighLat: -8.62, HighLng: 115.16}

// densePlaces simulates Google: a rectangle larger than maxSpan returns the 60-place maximum
// (3 pages of 20), a smaller one returns 7 places on a single page.
func densePlaces(maxSpan float64) *fakePlaces {
	f := &fakePlaces{}
	f.respond = func(tok string) ([]*model.BusinessLead, string, int, bool, error) {
		spec := f.specs[len(f.specs)-1]
		span := spec.Rect.High.Latitude - spec.Rect.Low.Latitude
		if span <= maxSpan {
			return leadsN(7), "", 1, true, nil
		}
		// dense: one page per call, three pages total
		page := 0
		if tok != "" {
			fmt.Sscanf(tok, "p%d", &page)
		}
		if page == 2 {
			return leadsN(20), "", 1, true, nil
		}
		return leadsN(20), fmt.Sprintf("p%d", page+1), 1, false, nil
	}
	return f
}

func areaKeyword() *model.ScraperKeyword {
	id := testArea.ID
	return &model.ScraperKeyword{ID: uuid.New(), Keyword: "villa · Canggu", Mode: model.KeywordModeArea, Term: "villa", IncludedType: "lodging", AreaID: &id, Status: model.KeywordStatusReady, CurrentPage: 1}
}

func TestAreaRun_SearchesInsideTheArea(t *testing.T) {
	places := densePlaces(1) // whole area is "small": 7 places, one page
	kws, leads := &fakeKeywords{}, &fakeLeads{}
	res, err := newTestRunner(places, kws, leads, &fakeJobs{}).Run(context.Background(), areaKeyword(), uuid.New(), KeywordRunOptions{MaxPages: 3})
	if err != nil {
		t.Fatal(err)
	}
	spec := places.specs[0]
	if spec.Query != "villa" || spec.IncludedType != "lodging" || spec.Rect == nil || spec.Rect.Low.Latitude != testArea.LowLat || spec.Rect.High.Longitude != testArea.HighLng {
		t.Errorf("unexpected spec %+v rect %+v", spec, spec.Rect)
	}
	if res.Status != model.KeywordStatusCompleted || len(leads.saved) != 7 || leads.saved[0].ScrapeKeywords[0] != "villa · Canggu" {
		t.Errorf("status=%s saved=%d", res.Status, len(leads.saved))
	}
	if len(kws.outcomes) != 1 || kws.outcomes[0] != 7 {
		t.Errorf("outcome not recorded: %v", kws.outcomes)
	}
}

func TestAreaRun_SplitsBusyAreaAcrossRuns(t *testing.T) {
	// The area is 0.05 deg tall; cells up to 0.02 deg are "small". Root (0.05) is dense ->
	// split into 0.025 cells, also dense -> split into 0.0125 cells, which return 7 each.
	places := densePlaces(0.02)
	kws, leads, jobs := &fakeKeywords{}, &fakeLeads{}, &fakeJobs{}
	r := newTestRunner(places, kws, leads, jobs)
	kw := areaKeyword()

	runs := 0
	for {
		runs++
		res, err := r.Run(context.Background(), kw, uuid.New(), KeywordRunOptions{MaxPages: 5})
		if err != nil {
			t.Fatal(err)
		}
		kw.Status = res.Status
		kw.CurrentPage = res.CurrentPage
		if res.Status != model.KeywordStatusPaginating || runs > 50 {
			break
		}
	}
	if kw.Status != model.KeywordStatusCompleted {
		t.Fatalf("ended with %s after %d runs", kw.Status, runs)
	}

	var split, done, maxDepth int
	for _, c := range kws.cells {
		switch c.Status {
		case "split":
			split++
		case "done":
			done++
		}
		if c.Depth > maxDepth {
			maxDepth = c.Depth
		}
	}
	// 1 root + 4 depth-1 cells split; 16 depth-2 leaves done.
	if split != 5 || done != 16 || maxDepth != 2 {
		t.Errorf("split=%d done=%d maxDepth=%d", split, done, maxDepth)
	}
	// 60 (root) + 4*60 (depth 1) + 16*7 = 412 results, all saved
	if len(leads.saved) != 60+240+112 {
		t.Errorf("saved %d leads", len(leads.saved))
	}
	// Each run respected its page budget.
	if len(places.calls) > runs*5 {
		t.Errorf("%d calls over %d runs exceeds 5 pages per run", len(places.calls), runs)
	}
}

func TestAreaRun_DepthLimitStopsSplitting(t *testing.T) {
	places := densePlaces(0) // every cell is dense
	kws := &fakeKeywords{}
	r := newTestRunner(places, kws, &fakeLeads{}, &fakeJobs{})
	kw := areaKeyword()
	for i := 0; i < 500; i++ {
		res, err := r.Run(context.Background(), kw, uuid.New(), KeywordRunOptions{MaxPages: 30})
		if err != nil {
			t.Fatal(err)
		}
		kw.Status = res.Status
		if res.Status != model.KeywordStatusPaginating {
			break
		}
	}
	if kw.Status != model.KeywordStatusCompleted {
		t.Fatalf("status %s", kw.Status)
	}
	for _, c := range kws.cells {
		if c.Depth > maxCellDepth || (c.Depth == maxCellDepth && c.Status != "done") {
			t.Fatalf("cell at depth %d has status %s", c.Depth, c.Status)
		}
	}
}

func TestAreaRun_EmptyAreaIsNoResults(t *testing.T) {
	places := &fakePlaces{respond: func(string) ([]*model.BusinessLead, string, int, bool, error) {
		return nil, "", 1, true, nil
	}}
	kws := &fakeKeywords{}
	res, err := newTestRunner(places, kws, &fakeLeads{}, &fakeJobs{}).Run(context.Background(), areaKeyword(), uuid.New(), KeywordRunOptions{MaxPages: 3})
	if err != nil || res.Status != model.KeywordStatusNoResults {
		t.Errorf("status=%s err=%v", res.Status, err)
	}
}

func TestAreaRun_QuotaMidRunKeepsEarlierCells(t *testing.T) {
	calls := 0
	places := &fakePlaces{}
	places.respond = func(tok string) ([]*model.BusinessLead, string, int, bool, error) {
		calls++
		if calls == 1 {
			return leadsN(20), "p1", 1, false, nil
		}
		return nil, "", 0, false, fmt.Errorf("%w: budget", gmaps.ErrQuotaExceeded)
	}
	kws, jobs := &fakeKeywords{}, &fakeJobs{}
	res, err := newTestRunner(places, kws, &fakeLeads{}, jobs).Run(context.Background(), areaKeyword(), uuid.New(), KeywordRunOptions{MaxPages: 3})
	if !gmaps.IsQuotaError(err) {
		t.Fatalf("expected quota error to be returned, got %v", err)
	}
	if res.Status != model.KeywordStatusPaginating || jobs.last().status != model.JobStatusCompleted || jobs.last().scraped != 20 {
		t.Errorf("status=%s job=%+v", res.Status, jobs.last())
	}
	if kws.cells[0].NextPageToken != "p1" || kws.cells[0].PagesFetched != 1 {
		t.Errorf("cell progress not kept: %+v", kws.cells[0])
	}
}

func TestAreaRun_NextCycleStartsFresh(t *testing.T) {
	places := densePlaces(1)
	kws := &fakeKeywords{}
	r := newTestRunner(places, kws, &fakeLeads{}, &fakeJobs{})
	kw := areaKeyword()
	res, _ := r.Run(context.Background(), kw, uuid.New(), KeywordRunOptions{MaxPages: 3})
	kw.Status = res.Status
	if _, err := r.Run(context.Background(), kw, uuid.New(), KeywordRunOptions{MaxPages: 3}); err != nil {
		t.Fatal(err)
	}
	if len(kws.cells) != 1 || kws.cells[0].Status != "done" || len(places.calls) != 2 {
		t.Errorf("cells=%d calls=%d", len(kws.cells), len(places.calls))
	}
}
