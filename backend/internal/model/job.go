package model

import (
	"time"

	"github.com/google/uuid"
)

type JobStatus string

const (
	JobStatusPending    JobStatus = "PENDING"
	JobStatusProcessing JobStatus = "PROCESSING"
	JobStatusCompleted  JobStatus = "COMPLETED"
	JobStatusFailed     JobStatus = "FAILED"
	JobStatusCancelled  JobStatus = "CANCELLED"
)

// JobKind says which pipeline runs a queued job.
type JobKind string

const (
	JobKindManual  JobKind = "manual"  // ad-hoc search from the New Job form (query + location + filters)
	JobKindKeyword JobKind = "keyword" // one cursor run of a saved keyword
)

type ScrapeJob struct {
	ID             uuid.UUID              `json:"id"`
	Status         JobStatus              `json:"status"`
	Query          string                 `json:"query"`
	Location       string                 `json:"location"`
	Category       string                 `json:"category"`
	MaxResults     int                    `json:"max_results"`
	EnrichContacts bool                   `json:"enrich_contacts"`
	TotalScraped   int                    `json:"total_scraped"`
	TotalEnriched  int                    `json:"total_enriched"`
	ErrorCount     int                    `json:"error_count"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	StartedAt      *time.Time             `json:"started_at,omitempty"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	Kind           JobKind                `json:"kind"`
	KeywordID      *uuid.UUID             `json:"keyword_id,omitempty"`
	Attempts       int                    `json:"attempts"`
}

type CreateJobRequest struct {
	Query                string   `json:"query"`
	Location             string   `json:"location"`
	Category             string   `json:"category"`
	Queries              []string `json:"queries,omitempty"`
	Locations            []string `json:"locations,omitempty"`
	Categories           []string `json:"categories,omitempty"`
	MaxResults           int      `json:"max_results"`
	EnrichContacts       bool     `json:"enrich_contacts"`
	Engine               string   `json:"engine,omitempty"`
	OnlyNewBusinesses    bool     `json:"only_new_businesses,omitempty"`
	RequireWebsite       bool     `json:"require_website,omitempty"`
	WebsiteFilter        string   `json:"website_filter,omitempty"`
	RecencyFilter        string   `json:"recency_filter,omitempty"`
	MaxReviewCountForNew int      `json:"max_review_count_for_new,omitempty"`
}
