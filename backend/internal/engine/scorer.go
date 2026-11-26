package engine

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"business-scraping-service/backend/internal/repository"
	"business-scraping-service/backend/internal/scoring"
)

// LeadScorer recalculates lead scores after leads change.
type LeadScorer interface {
	RescoreLeads(ctx context.Context, ids []uuid.UUID)
}

// Scorer stores every active scoring profile's score for each lead it applies to.
type Scorer struct {
	repo  *repository.ScoringRepository
	areas *repository.AreaRepository
	mu    sync.Mutex // one recalculation at a time keeps "remove stale scores" consistent
}

func NewScorer(repo *repository.ScoringRepository, areas *repository.AreaRepository) *Scorer {
	return &Scorer{repo: repo, areas: areas}
}

// RescoreLeads recalculates the given leads. Errors are logged: scoring never blocks scraping.
func (s *Scorer) RescoreLeads(ctx context.Context, ids []uuid.UUID) {
	if s == nil || len(ids) == 0 {
		return
	}
	if err := s.rescore(ctx, ids); err != nil {
		slog.Error("Failed to rescore leads", "count", len(ids), "error", err)
	}
}

// RescoreAll recalculates every lead (after a profile or area change, and at startup).
func (s *Scorer) RescoreAll(ctx context.Context) error {
	start := time.Now()
	if err := s.rescore(ctx, nil); err != nil {
		return err
	}
	slog.Info("Rescored all leads", "duration_ms", time.Since(start).Milliseconds())
	return nil
}

// RescoreAllInBackground runs RescoreAll detached from the caller.
func (s *Scorer) RescoreAllInBackground() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := s.RescoreAll(ctx); err != nil {
			slog.Error("Failed to rescore all leads", "error", err)
		}
	}()
}

func (s *Scorer) rescore(ctx context.Context, ids []uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	profiles, err := s.repo.ListProfiles(ctx, true)
	if err != nil {
		return err
	}
	type compiled struct {
		id        uuid.UUID
		appliesTo []string
		cfg       scoring.Config
	}
	var cps []compiled
	for _, p := range profiles {
		var cfg scoring.Config
		if err := json.Unmarshal(p.Config, &cfg); err != nil {
			slog.Error("Skipping scoring profile with invalid config", "profile", p.Name, "error", err)
			continue
		}
		cps = append(cps, compiled{p.ID, p.AppliesToTypes, cfg})
	}

	var areas []scoring.Area
	if s.areas != nil {
		list, err := s.areas.List(ctx)
		if err != nil {
			return err
		}
		for _, a := range list {
			areas = append(areas, scoring.Area{Name: a.Name, LowLat: a.LowLat, LowLng: a.LowLng, HighLat: a.HighLat, HighLng: a.HighLng})
		}
	}

	leads, err := s.repo.LeadsForScoring(ctx, ids)
	if err != nil {
		return err
	}

	var rows []repository.ScoreRow
	suggestions := map[uuid.UUID][2]string{}
	for _, l := range leads {
		sugType, sugName := scoring.SuggestGroup(l.Name, l.Category)
		if sugType != l.SuggestedGroupType || sugName != l.SuggestedGroupName {
			suggestions[l.ID] = [2]string{sugType, sugName}
		}
		groupType := sugType
		if l.GroupType != nil && scoring.ValidGroupType(*l.GroupType) {
			groupType = *l.GroupType
		}

		sl := scoring.Lead{
			Name: l.Name, Category: l.Category, Types: l.Types, Rating: l.Rating, ReviewCount: l.ReviewCount,
			Website: l.Website, Emails: l.Emails, Phone: l.PrimaryPhone,
			FacebookURL: l.FacebookURL, InstagramURL: l.InstagramURL, TikTokURL: l.TikTokURL,
			LinkedInURL: l.LinkedInURL, TwitterURL: l.TwitterURL, YouTubeURL: l.YouTubeURL,
			GroupType: groupType,
		}
		var area string
		areaDone := false
		for _, p := range cps {
			if !scoring.Applies(p.appliesTo, sl) {
				continue
			}
			if !areaDone {
				area = scoring.AreaFor(areas, l.Latitude, l.Longitude, l.Address)
				areaDone = true
			}
			res := scoring.Compute(sl, p.cfg, area)
			bd, err := json.Marshal(res.Breakdown)
			if err != nil {
				slog.Error("skipping lead score: breakdown not encodable", "lead_id", l.ID, "profile_id", p.id, "error", err)
				continue
			}
			rows = append(rows, repository.ScoreRow{
				LeadID: l.ID, ProfileID: p.id, Score: res.Score, RawScore: res.Raw,
				ComputedTier: res.Tier, AreaName: area, Breakdown: bd,
			})
		}
	}

	scope := ids
	if ids != nil {
		// Only leads that still exist.
		scope = make([]uuid.UUID, 0, len(leads))
		for _, l := range leads {
			scope = append(scope, l.ID)
		}
	}
	return s.repo.SaveScores(ctx, scope, rows, suggestions)
}
