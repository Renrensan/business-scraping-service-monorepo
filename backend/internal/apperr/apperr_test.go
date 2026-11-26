package apperr

import (
	"errors"
	"fmt"
	"testing"
)

func TestKindsSurviveWrapping(t *testing.T) {
	tests := []struct {
		name string
		err  error
		kind error
	}{
		{"invalid", Invalid("term is required"), ErrInvalid},
		{"not found", NotFound("area %s not found", "x"), ErrNotFound},
		{"conflict", Conflict("already exists"), ErrConflict},
		{"unavailable", Unavailable("paused until %s", "08:00"), ErrUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := fmt.Errorf("creating keyword: %w", tt.err)
			if !errors.Is(wrapped, tt.kind) || !IsUserFacing(wrapped) {
				t.Errorf("kind lost through wrapping: %v", wrapped)
			}
		})
	}
	if IsUserFacing(errors.New("pq: connection refused")) {
		t.Error("plain errors are server faults")
	}
	if got := Invalid("term is required").Error(); got != "term is required" {
		t.Errorf("message %q", got)
	}
}
