package gmaps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIsInvalidPageTokenError(t *testing.T) {
	if !IsInvalidPageTokenError(ErrInvalidPageToken) {
		t.Errorf("Expected ErrInvalidPageToken to be recognized")
	}

	wrapped := fmt.Errorf("places error: %w", ErrInvalidPageToken)
	if !IsInvalidPageTokenError(wrapped) {
		t.Errorf("Expected wrapped ErrInvalidPageToken to be recognized")
	}

	googleErr := errors.New("google places api error [INVALID_ARGUMENT]: SearchTextPageToken cannot be decoded.")
	if !IsInvalidPageTokenError(googleErr) {
		t.Errorf("Expected SearchTextPageToken cannot be decoded to be recognized")
	}

	matchErr := errors.New("Request parameters for paging requests must match the initial SearchText request")
	if !IsInvalidPageTokenError(matchErr) {
		t.Errorf("Expected paging requests must match error to be recognized")
	}

	randomErr := errors.New("network timeout")
	if IsInvalidPageTokenError(randomErr) {
		t.Errorf("Did not expect random network error to be recognized as invalid page token")
	}
}

func TestSearchPlacesTextWithCursor_PaginationFlow(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var req TextSearchRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if callCount == 1 {
			// Page 1
			if req.PageToken != "" {
				t.Errorf("Expected empty PageToken on page 1, got %q", req.PageToken)
			}
			resp := TextSearchResponse{
				Places: []map[string]interface{}{
					{"id": "places/place_1", "displayName": map[string]interface{}{"text": "Place 1"}},
					{"id": "places/place_2", "displayName": map[string]interface{}{"text": "Place 2"}},
				},
				NextPageToken: "token_for_page_2",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if callCount == 2 {
			// Page 2
			if req.PageToken != "token_for_page_2" {
				t.Errorf("Expected PageToken 'token_for_page_2' on page 2, got %q", req.PageToken)
			}
			resp := TextSearchResponse{
				Places: []map[string]interface{}{
					{"id": "places/place_3", "displayName": map[string]interface{}{"text": "Place 3"}},
				},
				NextPageToken: "", // End of results
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
	}))
	defer server.Close()

	client := NewPlacesAPIClient("dummy_api_key")
	client.httpClient = server.Client()
	client.baseURL = server.URL
	client.pagePacing = 1 * time.Millisecond

	ctx := context.Background()

	// Scrape 1: start with empty cursor, maxPages = 1, pageSize = 10
	leads1, nextToken1, pagesFetched1, cycleFinished1, err := client.SearchPlacesTextWithCursor(ctx, "dentist jakarta", "", 1, 10)
	if err != nil {
		t.Fatalf("Unexpected error on scrape 1: %v", err)
	}
	if len(leads1) != 2 {
		t.Errorf("Expected 2 leads on page 1, got %d", len(leads1))
	}
	if nextToken1 != "token_for_page_2" {
		t.Errorf("Expected nextToken 'token_for_page_2', got %q", nextToken1)
	}
	if pagesFetched1 != 1 {
		t.Errorf("Expected 1 page fetched, got %d", pagesFetched1)
	}
	if cycleFinished1 {
		t.Errorf("Expected cycleFinished to be false after page 1")
	}

	// Scrape 2: continue with nextToken1, maxPages = 1, pageSize = 10
	leads2, nextToken2, pagesFetched2, cycleFinished2, err := client.SearchPlacesTextWithCursor(ctx, "dentist jakarta", nextToken1, 1, 10)
	if err != nil {
		t.Fatalf("Unexpected error on scrape 2: %v", err)
	}
	if len(leads2) != 1 {
		t.Errorf("Expected 1 lead on page 2, got %d", len(leads2))
	}
	if nextToken2 != "" {
		t.Errorf("Expected nextToken to be empty, got %q", nextToken2)
	}
	if pagesFetched2 != 1 {
		t.Errorf("Expected 1 page fetched, got %d", pagesFetched2)
	}
	if !cycleFinished2 {
		t.Errorf("Expected cycleFinished to be true after final page")
	}
}

func TestSearchPlacesTextWithCursor_InvalidPageToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := TextSearchResponse{
			Error: &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
				Status  string `json:"status"`
			}{
				Code:    400,
				Message: "SearchTextPageToken cannot be decoded.",
				Status:  "INVALID_ARGUMENT",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewPlacesAPIClient("dummy_api_key")
	client.httpClient = server.Client()
	client.baseURL = server.URL
	client.pagePacing = 1 * time.Millisecond

	ctx := context.Background()
	_, _, _, _, err := client.SearchPlacesTextWithCursor(ctx, "dentist jakarta", "expired_token_xyz", 1, 10)
	if err == nil {
		t.Fatalf("Expected error for expired page token, got nil")
	}
	if !IsInvalidPageTokenError(err) {
		t.Errorf("Expected IsInvalidPageTokenError(err) to be true, got %v", err)
	}
}

func TestSearchPlacesTextWithCursor_MultiPageGracefulEndOfResults(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if callCount == 1 {
			// Page 1 succeeds
			resp := TextSearchResponse{
				Places: []map[string]interface{}{
					{"id": "places/place_1", "displayName": map[string]interface{}{"text": "Place 1"}},
					{"id": "places/place_2", "displayName": map[string]interface{}{"text": "Place 2"}},
				},
				NextPageToken: "token_page_2",
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		// Page 2 returns INVALID_ARGUMENT (end of results / expired)
		resp := TextSearchResponse{
			Error: &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
				Status  string `json:"status"`
			}{
				Code:    400,
				Message: "SearchTextPageToken cannot be decoded.",
				Status:  "INVALID_ARGUMENT",
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewPlacesAPIClient("dummy_api_key")
	client.httpClient = server.Client()
	client.baseURL = server.URL
	client.pagePacing = 1 * time.Millisecond

	ctx := context.Background()
	// Request 2 pages
	leads, nextToken, pagesFetched, cycleFinished, err := client.SearchPlacesTextWithCursor(ctx, "dentist jakarta", "", 2, 10)
	if err != nil {
		t.Fatalf("Expected graceful completion without error, got %v", err)
	}
	if len(leads) != 2 {
		t.Errorf("Expected 2 leads from page 1 to be preserved, got %d", len(leads))
	}
	if pagesFetched != 1 {
		t.Errorf("Expected 1 page fetched, got %d", pagesFetched)
	}
	if !cycleFinished {
		t.Errorf("Expected cycleFinished to be true")
	}
	if nextToken != "" {
		t.Errorf("Expected nextToken to be empty, got %q", nextToken)
	}
}
