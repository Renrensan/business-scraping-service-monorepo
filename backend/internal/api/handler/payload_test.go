package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Malformed bodies are rejected before any service is used (the handlers have nil services here,
// so reaching one would panic).
func TestMalformedBodiesAreRejected(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"trigger scrape", (&SystemHandler{}).TriggerImmediateScrape},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"queries": [`))
			tt.handler(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status %d, want 400", rec.Code)
			}
		})
	}
}
