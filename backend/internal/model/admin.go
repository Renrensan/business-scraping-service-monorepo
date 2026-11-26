package model

import (
	"time"

	"github.com/google/uuid"
)

type AdminUser struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

type LoginRequest struct {
	UsernameOrEmail string `json:"usernameOrEmail"`
	Password        string `json:"password"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}
