package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"

	"business-scraping-service/backend/internal/engine/enrichment"
	"business-scraping-service/backend/internal/engine/gmaps"
	"business-scraping-service/backend/internal/model"
)

// ---- fakes ----

type fakePlaces struct {
	calls   []string // page tokens requested
	respond func(token string) ([]*model.BusinessLead, string, int, bool, error)
}

func (f *fakePlaces) SearchPlacesText(ctx context.Context, q string, max int) ([]*model.BusinessLead, error) {
	return nil, errors.New("not used")
}

func (f *fakePlaces) SearchPlacesTextWithCursor(ctx context.Context, q, token string, maxPages, pageSize int) ([]*model.BusinessLead, string, int, bool, error) {
	f.calls = append(f.calls, token)
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
func (f *fakeJobs) IncrementProgress(ctx context.Context, id uuid.UUID, a, b, c int) error { return nil }
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
}

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
	r := NewKeywordRunner(kws, leads, jobs, nil, func() string { return "key" })
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
	r := NewKeywordRunner(kws, &fakeLeads{}, jobs, nil, func() string { return "" })
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
