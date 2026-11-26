package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"business-scraping-service/backend/internal/engine"
	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"
	"business-scraping-service/backend/internal/scoring"
)

type ScoringService struct {
	repo   *repository.ScoringRepository
	scorer *engine.Scorer
}

func NewScoringService(repo *repository.ScoringRepository, scorer *engine.Scorer) *ScoringService {
	return &ScoringService{repo: repo, scorer: scorer}
}

// ProfileInput is what an admin edits. Config is the formula (see scoring.Config).
type ProfileInput struct {
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	AppliesToTypes []string       `json:"applies_to_types"`
	Config         scoring.Config `json:"config"`
	IsActive       *bool          `json:"is_active"`
}

// ScoringOptions lists the choices the profile editor offers.
type ScoringOptions struct {
	Channels      []string       `json:"channels"`
	GroupTypes    []string       `json:"group_types"`
	Tiers         []string       `json:"tiers"`
	DefaultConfig scoring.Config `json:"default_config"`
}

func (s *ScoringService) Options() ScoringOptions {
	return ScoringOptions{
		Channels:      scoring.AllChannels,
		GroupTypes:    scoring.GroupTypes,
		Tiers:         []string{scoring.TierHot, scoring.TierGood, scoring.TierLow, scoring.TierSkip},
		DefaultConfig: scoring.GuestChatProConfig(),
	}
}

func validateProfile(in *ProfileInput) error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return fmt.Errorf("name is required")
	}
	c := &in.Config
	c.Normalize()
	for _, ch := range c.Channels {
		ok := false
		for _, known := range scoring.AllChannels {
			if ch == known {
				ok = true
			}
		}
		if !ok {
			return fmt.Errorf("unknown channel %q", ch)
		}
	}
	for g := range c.GroupBonuses {
		if !scoring.ValidGroupType(g) {
			return fmt.Errorf("unknown group type %q", g)
		}
	}
	check := func(where string, co scoring.Cutoff) error {
		if co.Good > co.Hot {
			return fmt.Errorf("%s: the GOOD cut-off (%d) must not be above HOT (%d)", where, co.Good, co.Hot)
		}
		return nil
	}
	if err := check("default", c.DefaultCutoff); err != nil {
		return err
	}
	for area, co := range c.AreaCutoffs {
		if strings.TrimSpace(area) == "" {
			return fmt.Errorf("area cut-off without an area name")
		}
		if err := check(area, co); err != nil {
			return err
		}
	}
	var types []string
	for _, t := range in.AppliesToTypes {
		if t = strings.TrimSpace(t); t != "" {
			types = append(types, t)
		}
	}
	in.AppliesToTypes = types
	if in.AppliesToTypes == nil {
		in.AppliesToTypes = []string{}
	}
	return nil
}

func (s *ScoringService) ListProfiles(ctx context.Context) ([]*model.ScoringProfile, error) {
	return s.repo.ListProfiles(ctx, false)
}

func (s *ScoringService) CreateProfile(ctx context.Context, in ProfileInput) (*model.ScoringProfile, error) {
	if err := validateProfile(&in); err != nil {
		return nil, err
	}
	cfg, _ := json.Marshal(in.Config)
	p := &model.ScoringProfile{Name: in.Name, Description: in.Description, AppliesToTypes: in.AppliesToTypes, Config: cfg, IsActive: in.IsActive == nil || *in.IsActive}
	if err := s.repo.CreateProfile(ctx, p); err != nil {
		return nil, err
	}
	if err := s.scorer.RescoreAll(ctx); err != nil {
		return nil, fmt.Errorf("profile saved but scoring failed: %w", err)
	}
	return s.repo.GetProfile(ctx, p.ID)
}

func (s *ScoringService) UpdateProfile(ctx context.Context, id uuid.UUID, in ProfileInput) (*model.ScoringProfile, error) {
	existing, err := s.repo.GetProfile(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := validateProfile(&in); err != nil {
		return nil, err
	}
	cfg, _ := json.Marshal(in.Config)
	existing.Name, existing.Description, existing.AppliesToTypes, existing.Config = in.Name, in.Description, in.AppliesToTypes, cfg
	if in.IsActive != nil {
		existing.IsActive = *in.IsActive
	}
	if err := s.repo.UpdateProfile(ctx, existing); err != nil {
		return nil, err
	}
	if err := s.scorer.RescoreAll(ctx); err != nil {
		return nil, fmt.Errorf("profile saved but scoring failed: %w", err)
	}
	return s.repo.GetProfile(ctx, id)
}

func (s *ScoringService) DeleteProfile(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteProfile(ctx, id)
}

func (s *ScoringService) Rescore(ctx context.Context) error {
	return s.scorer.RescoreAll(ctx)
}

// LeadGroupingInput sets a lead's group by hand; an empty string resets to the suggestion.
type LeadGroupingInput struct {
	GroupType *string `json:"group_type"`
	GroupName *string `json:"group_name"`
}

func (s *ScoringService) SetLeadGrouping(ctx context.Context, leadID uuid.UUID, in LeadGroupingInput) error {
	if in.GroupType != nil && *in.GroupType != "" && !scoring.ValidGroupType(*in.GroupType) {
		return repository.ErrInvalidGroupType
	}
	if err := s.repo.SetLeadGrouping(ctx, leadID, in.GroupType, in.GroupName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("lead not found")
		}
		return err
	}
	s.scorer.RescoreLeads(ctx, []uuid.UUID{leadID})
	return nil
}

// ScoreOverrideInput sets a tier by hand (empty = back to the computed tier) and a "Why" note.
type ScoreOverrideInput struct {
	TierOverride *string `json:"tier_override"`
	Note         string  `json:"note"`
}

func (s *ScoringService) SetScoreOverride(ctx context.Context, leadID, profileID uuid.UUID, in ScoreOverrideInput) error {
	if in.TierOverride != nil && *in.TierOverride != "" && !scoring.ValidTier(*in.TierOverride) {
		return repository.ErrInvalidTier
	}
	return s.repo.SetScoreOverride(ctx, leadID, profileID, in.TierOverride, in.Note)
}

func (s *ScoringService) ScoresForLead(ctx context.Context, leadID uuid.UUID) ([]model.LeadScore, error) {
	return s.repo.ScoresForLead(ctx, leadID)
}
