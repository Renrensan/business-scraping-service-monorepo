package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"business-scraping-service/backend/internal/api/middleware"
	"business-scraping-service/backend/internal/apperr"
	"business-scraping-service/backend/internal/engine"
	"business-scraping-service/backend/internal/repository"
)

func TestRespondErr(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantText   string // must appear in the response
		hidden     string // must not appear in the response
	}{
		{"database fault is hidden", errors.New(`pq: relation "business_leads" does not exist`), 500, "Failed to load (reference req-1)", "business_leads"},
		{"wrapped fault is hidden", fmt.Errorf("loading keyword: %w", errors.New("dial tcp 10.0.0.5:5432: refused")), 500, "Failed to load", "10.0.0.5"},
		{"validation message is shown", apperr.Invalid("term is required"), 400, "term is required", ""},
		{"not found", repository.ErrProfileNotFound, 404, "scoring profile not found", ""},
		{"conflict through wrapping", fmt.Errorf("keyword %q: %w", "villa", engine.ErrKeywordBusy), 409, "already queued", ""},
		{"quota pause", apperr.Unavailable("scraping is paused until 08:00"), 429, "paused until 08:00", ""},
		{"missing api key", engine.ErrNoAPIKey, 400, "api key is not configured", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDKey, "req-1"))

			respondErr(rec, req, tt.err, "Failed to load")

			var body struct {
				Error string `json:"error"`
			}
			_ = json.Unmarshal(rec.Body.Bytes(), &body)
			if rec.Code != tt.wantStatus || !strings.Contains(body.Error, tt.wantText) {
				t.Errorf("got %d %q, want %d containing %q", rec.Code, body.Error, tt.wantStatus, tt.wantText)
			}
			if tt.hidden != "" && strings.Contains(rec.Body.String(), tt.hidden) {
				t.Errorf("response leaks %q: %s", tt.hidden, rec.Body.String())
			}
		})
	}
}
