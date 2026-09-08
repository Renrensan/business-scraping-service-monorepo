package model

import (
	"time"

	"github.com/google/uuid"
)

type SessionData struct {
	AdminID      uuid.UUID `json:"admin_id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	CreatedAt    time.Time `json:"created_at"`
	LastActiveAt time.Time `json:"last_active_at"`
}
