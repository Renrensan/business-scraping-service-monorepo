package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AuditEntry is one recorded admin action.
type AuditEntry struct {
	ID         int64           `json:"id"`
	CreatedAt  time.Time       `json:"created_at"`
	AdminID    *uuid.UUID      `json:"admin_id,omitempty"`
	AdminName  string          `json:"admin_name"`
	Action     string          `json:"action"`
	Summary    string          `json:"summary"`
	EntityType string          `json:"entity_type,omitempty"`
	EntityID   string          `json:"entity_id,omitempty"`
	EntityName string          `json:"entity_name,omitempty"` // resolved when listing (e.g. the lead's name)
	Method     string          `json:"method,omitempty"`
	Route      string          `json:"route,omitempty"`
	Status     int             `json:"status"`
	Details    json.RawMessage `json:"details,omitempty"`
	IP         string          `json:"ip,omitempty"`
	RequestID  string          `json:"request_id,omitempty"`
}

// AuditFilter selects audit entries, newest first.
type AuditFilter struct {
	AdminID    *uuid.UUID
	EntityType string
	EntityID   string
	Action     string
	Search     string
	FailedOnly bool
	Page       int
	PageSize   int
}

type AuditListResponse struct {
	Items      []AuditEntry `json:"items"`
	TotalCount int64        `json:"total_count"`
	Page       int          `json:"page"`
	PageSize   int          `json:"page_size"`
	TotalPages int          `json:"total_pages"`
}
