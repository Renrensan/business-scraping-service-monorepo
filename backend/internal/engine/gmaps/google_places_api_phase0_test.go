package gmaps

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestClient(t *testing.T, h http.HandlerFunc) *PlacesAPIClient {
	t.Helper()
	server := httptest.NewServer(h)
	t.Cleanup(server.Close)
	c := NewPlacesAPIClient("dummy_api_key")
	c.httpClient = server.Client()
	c.baseURL = server.URL
	c.pagePacing = time.Millisecond
	return c
}

func TestInferIncludedType(t *testing.T) {
	cases := map[string]string{
		"Villa in Jimbaran, Bali":    "lodging",
		"villas canggu":              "lodging",
		"Guest House in Ubud":        "lodging",
		"homestay sidemen":           "lodging",
		"Hotel in Kuta, Bali":        "",
		"Spa in Denpasar, Bali":      "",
		"Villanova pizza":            "",
		"Digital Printing in Badung": "",
	}
	for q, want := range cases {
		if got := InferIncludedType(q); got != want {
			t.Errorf("InferIncludedType(%q) = %q, want %q", q, got, want)
		}
	}
}

func TestSearchPlacesTextWithCursor_SendsSameIncludedTypeOnEveryPage(t *testing.T) {
	calls := 0
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		var req TextSearchRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.IncludedType != "lodging" {
			t.Errorf("call %d: includedType = %q, want lodging", calls, req.IncludedType)
		}
		resp := TextSearchResponse{Places: []map[string]interface{}{{"id": "places/p"}}}
		if calls == 1 {
			resp.NextPageToken = "tok2"
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	leads, _, pages, finished, err := c.SearchPlacesTextWithCursor(context.Background(), "Villa in Jimbaran, Bali", "", 3, 20)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || pages != 2 || len(leads) != 2 || !finished {
		t.Errorf("calls=%d pages=%d leads=%d finished=%v", calls, pages, len(leads), finished)
	}
}

func TestSearchPlacesTextWithCursor_EmptyFirstPage(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`)) // what Google returns for "Villa in Jimbaran, Bali" without a type
	})
	leads, next, pages, finished, err := c.SearchPlacesTextWithCursor(context.Background(), "coworking in nowhere", "", 3, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(leads) != 0 || next != "" || pages != 1 || !finished {
		t.Errorf("leads=%d next=%q pages=%d finished=%v", len(leads), next, pages, finished)
	}
}

func TestSearchPlacesTextWithCursor_QuotaErrorIsTypedAndNotRetried(t *testing.T) {
	calls := 0
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":429,"message":"Quota exceeded for quota metric 'SearchTextRequest'","status":"RESOURCE_EXHAUSTED"}}`))
	})
	_, _, _, _, err := c.SearchPlacesTextWithCursor(context.Background(), "hotel", "some_token", 3, 20)
	if !IsQuotaError(err) {
		t.Fatalf("expected quota error, got %v", err)
	}
	if IsInvalidPageTokenError(err) {
		t.Errorf("quota error must not be treated as an invalid page token (would reset the cursor)")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retry on quota), got %d", calls)
	}
}

func TestSearchPlacesText_KeepsEarlierPagesOnLaterError(t *testing.T) {
	calls := 0
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			_ = json.NewEncoder(w).Encode(TextSearchResponse{Places: []map[string]interface{}{{"id": "places/a"}}, NextPageToken: "t"})
			return
		}
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":429,"message":"Quota exceeded","status":"RESOURCE_EXHAUSTED"}}`))
	})
	leads, err := c.SearchPlacesText(context.Background(), "hotel", 60)
	if !IsQuotaError(err) || len(leads) != 1 {
		t.Errorf("leads=%d err=%v", len(leads), err)
	}
}
