package model

import (
	"time"

	"github.com/google/uuid"
)

type EnrichmentStatus string

const (
	EnrichmentNone    EnrichmentStatus = "NONE"
	EnrichmentPending EnrichmentStatus = "PENDING"
	EnrichmentSuccess EnrichmentStatus = "SUCCESS"
	EnrichmentFailed  EnrichmentStatus = "FAILED"
)

// LeadStatus is the admin-assigned temperature of a contacted lead.
// Leads that have not been contacted have no status (nil).
type LeadStatus string

const (
	LeadStatusCold LeadStatus = "cold" // contacted, no response yet
	LeadStatusWarm LeadStatus = "warm" // they responded
	LeadStatusHot  LeadStatus = "hot"  // they look interested in the product
)

func (s LeadStatus) Valid() bool {
	switch s {
	case LeadStatusCold, LeadStatusWarm, LeadStatusHot:
		return true
	}
	return false
}

type BusinessLead struct {
	ID               uuid.UUID        `json:"id"`
	JobID            *uuid.UUID       `json:"job_id,omitempty"`
	PlaceID          string           `json:"place_id"`
	CID              string           `json:"cid"`
	Name             string           `json:"name"`
	Category         string           `json:"category"`
	Subcategories    []string         `json:"subcategories"`   // Google's own place types only
	ScrapeKeywords   []string         `json:"scrape_keywords"` // search keywords that returned this lead
	Address          string           `json:"address"`
	Street           string           `json:"street"`
	City             string           `json:"city"`
	State            string           `json:"state"`
	PostalCode       string           `json:"postal_code"`
	Country          string           `json:"country"`
	Latitude         float64          `json:"latitude"`
	Longitude        float64          `json:"longitude"`
	Rating           float64          `json:"rating"`
	ReviewCount      int              `json:"review_count"`
	PriceLevel       string           `json:"price_level"`
	Claimed          bool             `json:"claimed"`
	BusinessStatus   string           `json:"business_status"`
	Website          string           `json:"website"`
	PrimaryPhone     string           `json:"primary_phone"`
	Emails           []string         `json:"emails"`
	Phones           []string         `json:"phones"`
	FacebookURL      string           `json:"facebook_url"`
	InstagramURL     string           `json:"instagram_url"`
	LinkedinURL      string           `json:"linkedin_url"`
	TwitterURL       string           `json:"twitter_url"`
	YoutubeURL       string           `json:"youtube_url"`
	TiktokURL        string           `json:"tiktok_url"`
	GoogleMapsURL    string           `json:"google_maps_url"`
	OpeningHours     map[string]any   `json:"opening_hours,omitempty"`
	Photos           []string         `json:"photos,omitempty"`
	EnrichmentStatus EnrichmentStatus `json:"enrichment_status"`
	LastScrapedAt    time.Time        `json:"last_scraped_at"`
	LastEnrichedAt   *time.Time       `json:"last_enriched_at,omitempty"`
	Contacted        bool             `json:"contacted"`
	ContactedAt      *time.Time       `json:"contacted_at,omitempty"`
	ContactedBy      *uuid.UUID       `json:"contacted_by,omitempty"`
	ContactedByName  string           `json:"contacted_by_name,omitempty"` // username of the admin who marked it contacted
	LeadStatus       *LeadStatus      `json:"lead_status"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`

	// Grouping for scoring: set by hand (nil = use the suggestion from the name).
	GroupType          *string `json:"group_type"`
	GroupName          *string `json:"group_name"`
	SuggestedGroupType string  `json:"suggested_group_type"`
	SuggestedGroupName string  `json:"suggested_group_name"`

	// Score for the profile selected in a list request; all profiles' scores on the detail view.
	Score  *LeadScore  `json:"score,omitempty"`
	Scores []LeadScore `json:"scores,omitempty"`

	// People recorded by admins: the first-added one and the total on list rows, all on detail.
	MainContactName string        `json:"main_contact_name,omitempty"`
	MainContactRole string        `json:"main_contact_role,omitempty"`
	ContactCount    int           `json:"contact_count"`
	Contacts        []LeadContact `json:"contacts,omitempty"`
}

type BusinessFilter struct {
	Search     string
	Category   string
	City       string
	State      string
	HasEmail   *bool
	HasPhone   *bool
	HasWebsite *bool
	MinRating  *float64
	Claimed    *bool
	Contacted  *bool
	JobID      *uuid.UUID
	Keywords   []string
	Statuses   []LeadStatus
	OnlyNew    *bool
	MaxReviews *int
	// Scoring: rank by a profile, optionally only some tiers / one scoring area.
	ScoreProfileID   *uuid.UUID
	ScoreProfileName string // used when ScoreProfileID is not set
	Tiers            []string
	ScoredOnly       bool // only leads the profile applies to (have a score)
	ScoreArea        string

	SortBy    string
	SortOrder string
	Page      int
	PageSize  int
}

type BusinessListResponse struct {
	Items      []*BusinessLead `json:"items"`
	TotalCount int64           `json:"total_count"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}
