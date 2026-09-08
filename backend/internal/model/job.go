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
}

type CreateJobRequest struct {
	Query          string `json:"query"`
	Location       string `json:"location"`
	Category       string `json:"category"`
	MaxResults     int    `json:"max_results"`
	EnrichContacts bool   `json:"enrich_contacts"`
	Engine         string `json:"engine,omitempty"`
	ApiKey         string `json:"api_key,omitempty"`
}
