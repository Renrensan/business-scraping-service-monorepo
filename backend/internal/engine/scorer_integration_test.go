package engine

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"

	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"
	"business-scraping-service/backend/internal/scoring"
)

// Needs a disposable Postgres: TEST_DATABASE_URL=... (wiped).
func scoringDB(t *testing.T) (*repository.BusinessRepository, *repository.ScoringRepository, *Scorer, func(string, ...interface{})) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := repository.InitDB(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	exec := func(sql string, args ...interface{}) {
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	exec(`TRUNCATE business_leads, scrape_jobs, scraper_keywords CASCADE`)
	exec(`DELETE FROM scoring_profiles WHERE name <> 'GuestChatPro'`)
	exec(`UPDATE scoring_profiles SET is_active = true`)
	repo := repository.NewScoringRepository(pool)
	return repository.NewBusinessRepository(pool), repo, NewScorer(repo, repository.NewAreaRepository(pool)), exec
}

func lodging(name string, lat, lng float64, reviews int, rating float64) *model.BusinessLead {
	return &model.BusinessLead{
		Name: name, Address: name + " st", Category: "Hotel", Subcategories: []string{"Lodging", "Hotel"},
		Latitude: lat, Longitude: lng, ReviewCount: reviews, Rating: rating,
	}
}

func TestScorer_EndToEnd(t *testing.T) {
	leads, repo, scorer, exec := scoringDB(t)
	ctx := context.Background()

	// Doc example: Alaya Resort Ubud (inside the Ubud preset area).
	alaya := lodging("Alaya Resort Ubud", -8.507, 115.262, 4186, 4.7)
	alaya.Emails = []string{"info@alayaresorts.com"}
	alaya.Website = "https://alayaresorts.com"
	alaya.FacebookURL = "https://facebook.com/alayaubud"
	alaya.TiktokURL = "https://tiktok.com/@alayaubud"
	alaya.InstagramURL = "https://instagram.com/p/abc/" // broken: must not count
	// A sister property, scored low on its own, and an unrelated mid-scoring hotel.
	sister := lodging("Alaya Dedari Kumbara", -8.505, 115.265, 50, 4.6)
	mid := lodging("Ubud Hideaway", -8.508, 115.26, 3000, 4.6)
	mid.Website = "https://ubudhideaway.com"
	chain := lodging("Swiss-Belexpress Kuta Legian", -8.728, 115.172, 11000, 4.3)
	spa := &model.BusinessLead{Name: "Lotus Spa", Address: "x", Category: "Spa", Subcategories: []string{"Spa"}, ReviewCount: 900, Rating: 4.8}
	for _, l := range []*model.BusinessLead{alaya, sister, mid, chain, spa} {
		if _, err := leads.Upsert(ctx, l); err != nil {
			t.Fatal(err)
		}
	}

	if err := scorer.RescoreAll(ctx); err != nil {
		t.Fatal(err)
	}
	gcp, err := repo.GetProfileByName(ctx, "guestchatpro")
	if err != nil {
		t.Fatal(err)
	}

	scores, _ := repo.ScoresForLead(ctx, alaya.ID)
	if len(scores) != 1 || scores[0].Score != 74 || scores[0].Tier != scoring.TierHot || scores[0].AreaName != "Ubud" {
		t.Fatalf("alaya: %+v", scores)
	}
	var bd scoring.Breakdown
	_ = json.Unmarshal(scores[0].Breakdown, &bd)
	if bd.GroupType != scoring.GroupGroup || len(bd.Ignored) != 1 || bd.Ignored[0].Channel != "instagram" {
		t.Errorf("breakdown: %+v", bd)
	}
	if s, _ := repo.ScoresForLead(ctx, spa.ID); len(s) != 0 {
		t.Errorf("spa should not get a GuestChatPro score: %+v", s)
	}
	if s, _ := repo.ScoresForLead(ctx, chain.ID); len(s) != 1 || s[0].Tier != scoring.TierLow || s[0].AreaName != "Kuta" {
		t.Errorf("chain: %+v", s)
	}
	lead, _ := leads.GetByID(ctx, chain.ID)
	if lead.SuggestedGroupType != scoring.GroupChain || lead.SuggestedGroupName != "Swiss-Belhotel" {
		t.Errorf("suggestion not stored: %s/%s", lead.SuggestedGroupType, lead.SuggestedGroupName)
	}

	// Ranked list: Alaya first, its sister right after it (out of score order), then the rest.
	list, err := leads.List(ctx, model.BusinessFilter{ScoreProfileName: "GuestChatPro", SortBy: "score", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, l := range list.Items {
		order = append(order, l.Name)
	}
	want := []string{"Alaya Resort Ubud", "Alaya Dedari Kumbara", "Ubud Hideaway", "Swiss-Belexpress Kuta Legian", "Lotus Spa"}
	if len(order) != len(want) {
		t.Fatalf("order %v", order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order %v, want %v", order, want)
		}
	}
	if list.Items[4].Score != nil {
		t.Error("unscored lead should have no score")
	}

	// Tier filter and area filter.
	hot, _ := leads.List(ctx, model.BusinessFilter{ScoreProfileID: &gcp.ID, Tiers: []string{"hot"}, Page: 1, PageSize: 10})
	if hot.TotalCount != 1 || hot.Items[0].ID != alaya.ID {
		t.Errorf("hot filter: %d", hot.TotalCount)
	}
	kuta, _ := leads.List(ctx, model.BusinessFilter{ScoreProfileID: &gcp.ID, ScoreArea: "kuta", Page: 1, PageSize: 10})
	if kuta.TotalCount != 1 {
		t.Errorf("area filter: %d", kuta.TotalCount)
	}

	// Manual override and note survive a rescore; manual group changes the score.
	skip := "skip"
	if err := repo.SetScoreOverride(ctx, mid.ID, gcp.ID, &skip, "Already in Prospects tab"); err != nil {
		t.Fatal(err)
	}
	small := scoring.GroupSmallGroup
	name := "Hideaway Group"
	if err := repo.SetLeadGrouping(ctx, mid.ID, &small, &name); err != nil {
		t.Fatal(err)
	}
	scorer.RescoreLeads(ctx, []uuid.UUID{mid.ID})
	s, _ := repo.ScoresForLead(ctx, mid.ID)
	if s[0].Tier != "skip" || s[0].ComputedTier == "skip" || s[0].Note != "Already in Prospects tab" {
		t.Errorf("override lost: %+v", s[0])
	}
	var bd2 scoring.Breakdown
	_ = json.Unmarshal(s[0].Breakdown, &bd2)
	if bd2.GroupBonus != 8 {
		t.Errorf("manual group not applied: %+v", bd2)
	}
	if err := repo.SetScoreOverride(ctx, spa.ID, gcp.ID, &skip, ""); err != repository.ErrScoreNotFound {
		t.Errorf("override on unscored lead: %v", err)
	}

	// Deactivating the profile removes its scores.
	exec(`UPDATE scoring_profiles SET is_active = false WHERE id = $1`, gcp.ID)
	_ = scorer.RescoreAll(ctx)
	if s, _ := repo.ScoresForLead(ctx, alaya.ID); len(s) != 0 {
		t.Errorf("inactive profile still scored: %+v", s)
	}
}

func TestScorer_SecondProfile(t *testing.T) {
	leads, repo, scorer, _ := scoringDB(t)
	ctx := context.Background()
	spa := &model.BusinessLead{Name: "Lotus Spa", Address: "x", Category: "Spa", Subcategories: []string{"Spa"}, ReviewCount: 100, Rating: 4.5, Website: "lotus.com"}
	_, _ = leads.Upsert(ctx, spa)

	cfg := scoring.GuestChatProConfig()
	cfg.Channels = []string{scoring.ChannelWebsite, scoring.ChannelPhone}
	raw, _ := json.Marshal(cfg)
	p := &model.ScoringProfile{Name: "SpaPro", AppliesToTypes: []string{"Spa"}, Config: raw, IsActive: true}
	if err := repo.CreateProfile(ctx, p); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateProfile(ctx, &model.ScoringProfile{Name: "spapro", Config: raw}); err != repository.ErrProfileExists {
		t.Errorf("duplicate name: %v", err)
	}
	_ = scorer.RescoreAll(ctx)
	s, _ := repo.ScoresForLead(ctx, spa.ID)
	// 20 (reviews) + 5 (rating) + 4 (website) = 29
	if len(s) != 1 || s[0].ProfileName != "SpaPro" || s[0].Score != 29 {
		t.Errorf("spa profile: %+v", s)
	}
	if err := repo.DeleteProfile(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if s, _ := repo.ScoresForLead(ctx, spa.ID); len(s) != 0 {
		t.Errorf("scores not removed with profile")
	}
}
