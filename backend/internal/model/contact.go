package model

import (
	"time"

	"github.com/google/uuid"
)

// LeadContact is a person at a lead's business (owner, GM, reservations, ...).
type LeadContact struct {
	ID        uuid.UUID `json:"id"`
	LeadID    uuid.UUID `json:"lead_id"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LeadContactInput is what an admin enters for a contact.
type LeadContactInput struct {
	Name  string `json:"name"`
	Role  string `json:"role"`
	Phone string `json:"phone"`
	Email string `json:"email"`
	Note  string `json:"note"`
}
