package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ScoringProfile is an admin-defined way to rank leads (e.g. GuestChatPro). Config holds the
// formula (see package scoring).
type ScoringProfile struct {
	ID             uuid.UUID       `json:"id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	AppliesToTypes []string        `json:"applies_to_types"`
	Config         json.RawMessage `json:"config"`
	IsActive       bool            `json:"is_active"`
	ScoredLeads    int             `json:"scored_leads"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// ProfileSummary counts a profile's leads by tier, overall and per area, for its prospects page.
type ProfileSummary struct {
	ProfileID uuid.UUID      `json:"profile_id"`
	Total     int            `json:"total"`
	Tiers     map[string]int `json:"tiers"`
	Areas     []AreaTierInfo `json:"areas"`
}

type AreaTierInfo struct {
	Name  string         `json:"name"` // "" = leads outside every known area (default cut-offs)
	Total int            `json:"total"`
	Tiers map[string]int `json:"tiers"`
}

// LeadScore is a lead's score and tier under one profile.
type LeadScore struct {
	ProfileID    uuid.UUID       `json:"profile_id"`
	ProfileName  string          `json:"profile_name,omitempty"`
	Score        int             `json:"score"`
	RawScore     float64         `json:"raw_score"`
	Tier         string          `json:"tier"`
	ComputedTier string          `json:"computed_tier"`
	TierOverride *string         `json:"tier_override"`
	Note         string          `json:"note"`
	AreaName     string          `json:"area_name"`
	Breakdown    json.RawMessage `json:"breakdown"`
	ComputedAt   time.Time       `json:"computed_at"`
}
