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

type BusinessLead struct {
	ID               uuid.UUID              `json:"id"`
	JobID            *uuid.UUID             `json:"job_id,omitempty"`
	PlaceID          string                 `json:"place_id"`
	CID              string                 `json:"cid"`
	Name             string                 `json:"name"`
	Category         string                 `json:"category"`
	Subcategories    []string               `json:"subcategories"`
	Address          string                 `json:"address"`
	Street           string                 `json:"street"`
	City             string                 `json:"city"`
	State            string                 `json:"state"`
	PostalCode       string                 `json:"postal_code"`
	Country          string                 `json:"country"`
	Latitude         float64                `json:"latitude"`
	Longitude        float64                `json:"longitude"`
	Rating           float64                `json:"rating"`
	ReviewCount      int                    `json:"review_count"`
	PriceLevel       string                 `json:"price_level"`
	Claimed          bool                   `json:"claimed"`
	BusinessStatus   string                 `json:"business_status"`
	Website          string                 `json:"website"`
	PrimaryPhone     string                 `json:"primary_phone"`
	Emails           []string               `json:"emails"`
	Phones           []string               `json:"phones"`
	FacebookURL      string                 `json:"facebook_url"`
	InstagramURL     string                 `json:"instagram_url"`
	LinkedinURL      string                 `json:"linkedin_url"`
	TwitterURL       string                 `json:"twitter_url"`
	YoutubeURL       string                 `json:"youtube_url"`
	TiktokURL        string                 `json:"tiktok_url"`
	GoogleMapsURL    string                 `json:"google_maps_url"`
	OpeningHours     map[string]interface{} `json:"opening_hours,omitempty"`
	Photos           []string               `json:"photos,omitempty"`
	EnrichmentStatus EnrichmentStatus       `json:"enrichment_status"`
	LastScrapedAt    time.Time              `json:"last_scraped_at"`
	LastEnrichedAt   *time.Time             `json:"last_enriched_at,omitempty"`
	Contacted        bool                   `json:"contacted"`
	ContactedAt      *time.Time             `json:"contacted_at,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
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
	Keyword    string
	OnlyNew    *bool
	MaxReviews *int
	SortBy     string
	SortOrder  string
	Page       int
	PageSize   int
}

type BusinessListResponse struct {
	Items      []*BusinessLead `json:"items"`
	TotalCount int64           `json:"total_count"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}
