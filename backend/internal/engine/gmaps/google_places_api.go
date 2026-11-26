package gmaps

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"business-scraping-service/backend/internal/model"
)

var (
	// ErrInvalidPageToken means Google rejected the page token (expired, or the request no longer matches it).
	ErrInvalidPageToken = errors.New("invalid or expired page token")
	// ErrQuotaExceeded means the Places API quota (usually the daily SearchTextRequest limit) is used up.
	// Retrying is pointless until the quota resets, so callers should stop the whole run.
	ErrQuotaExceeded = errors.New("google places api quota exceeded")
)

func IsInvalidPageTokenError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrInvalidPageToken) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "pagetoken cannot be decoded") ||
		strings.Contains(msg, "paging requests must match") ||
		strings.Contains(msg, "invalid page token")
}

func IsQuotaError(err error) bool {
	return err != nil && errors.Is(err, ErrQuotaExceeded)
}

const textSearchFieldMask = "places.id,places.displayName,places.formattedAddress,places.location,places.types,places.primaryType,places.websiteUri,places.nationalPhoneNumber,places.rating,places.userRatingCount,places.businessStatus,places.googleMapsUri,places.regularOpeningHours,places.editorialSummary,places.reviews,nextPageToken"

type PlacesAPIClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	pagePacing time.Duration
}

func NewPlacesAPIClient(apiKey string) *PlacesAPIClient {
	return &PlacesAPIClient{
		apiKey:  apiKey,
		baseURL: "https://places.googleapis.com",
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		pagePacing: 2500 * time.Millisecond,
	}
}

type TextSearchRequest struct {
	TextQuery    string  `json:"textQuery"`
	IncludedType string  `json:"includedType,omitempty"`
	PageSize     int     `json:"pageSize,omitempty"`
	PageToken    string  `json:"pageToken,omitempty"`
	MinRating    float64 `json:"minRating,omitempty"`
}

type TextSearchResponse struct {
	Places        []map[string]interface{} `json:"places"`
	NextPageToken string                   `json:"nextPageToken,omitempty"`
	Error         *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

// newTextSearchRequest builds the request for a query. Google requires every page request to repeat
// the parameters of the first one, so the included type is derived from the query the same way each time.
func newTextSearchRequest(query string, pageSize int, pageToken string) TextSearchRequest {
	return TextSearchRequest{
		TextQuery:    query,
		IncludedType: InferIncludedType(query),
		PageSize:     pageSize,
		PageToken:    pageToken,
	}
}

// doTextSearch sends one Text Search (New) request and turns Google's errors into typed errors:
// ErrQuotaExceeded for quota/rate limits, ErrInvalidPageToken for rejected page tokens.
func (c *PlacesAPIClient) doTextSearch(ctx context.Context, reqBody TextSearchRequest) (*TextSearchResponse, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://places.googleapis.com"
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/places:searchText", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Goog-Api-Key", c.apiKey)
	httpReq.Header.Set("X-Goog-FieldMask", textSearchFieldMask)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call places api: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read places api response: %w", err)
	}

	var apiResp TextSearchResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, fmt.Errorf("%w: http 429", ErrQuotaExceeded)
		}
		return nil, fmt.Errorf("failed to decode places api response (http %d): %w", resp.StatusCode, err)
	}

	if apiResp.Error == nil && resp.StatusCode >= 400 {
		return nil, fmt.Errorf("google places api returned http %d", resp.StatusCode)
	}

	if apiResp.Error != nil {
		status := apiResp.Error.Status
		msg := apiResp.Error.Message
		lowerMsg := strings.ToLower(msg)

		if resp.StatusCode == http.StatusTooManyRequests || status == "RESOURCE_EXHAUSTED" || strings.Contains(lowerMsg, "quota exceeded") {
			return nil, fmt.Errorf("%w: [%s] %s", ErrQuotaExceeded, status, msg)
		}

		isTokenErr := reqBody.PageToken != "" &&
			(status == "INVALID_ARGUMENT" || strings.Contains(lowerMsg, "pagetoken") || strings.Contains(lowerMsg, "paging"))
		if isTokenErr {
			return nil, fmt.Errorf("%w: google places api [%s]: %s", ErrInvalidPageToken, status, msg)
		}

		return nil, fmt.Errorf("google places api error [%s]: %s", status, msg)
	}

	return &apiResp, nil
}

// SearchPlacesText executes a Text Search (New), following page tokens until maxResults is reached
// or Google has no more pages (Text Search returns at most 60 places per query).
func (c *PlacesAPIClient) SearchPlacesText(ctx context.Context, query string, maxResults int) ([]*model.BusinessLead, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("google api key is not configured")
	}

	pageSize := 20
	if maxResults > 0 && maxResults < 20 {
		pageSize = maxResults
	}

	var allLeads []*model.BusinessLead
	pageToken := ""

	for {
		apiResp, err := c.doTextSearch(ctx, newTextSearchRequest(query, pageSize, pageToken))
		if err != nil {
			// Keep what earlier pages returned rather than throwing it away.
			if len(allLeads) > 0 {
				return allLeads, err
			}
			return nil, err
		}

		for _, raw := range apiResp.Places {
			allLeads = append(allLeads, ParseRawPlaceData(raw))
			if maxResults > 0 && len(allLeads) >= maxResults {
				return allLeads, nil
			}
		}

		if apiResp.NextPageToken == "" {
			break
		}

		pageToken = apiResp.NextPageToken
		if err := sleepCtx(ctx, c.pacing()); err != nil {
			return allLeads, err
		}
	}

	return allLeads, nil
}

// SearchPlacesTextWithCursor executes a paginated Text Search starting from a given pageToken, traversing up to maxPages.
// It returns the leads, the token for the next page, the number of pages fetched and whether Google has no more pages.
func (c *PlacesAPIClient) SearchPlacesTextWithCursor(ctx context.Context, query string, startPageToken string, maxPages int, pageSize int) ([]*model.BusinessLead, string, int, bool, error) {
	if c.apiKey == "" {
		return nil, "", 0, false, fmt.Errorf("google api key is not configured")
	}

	if pageSize <= 0 || pageSize > 20 {
		pageSize = 20
	}
	if maxPages <= 0 {
		maxPages = 3
	}

	var allLeads []*model.BusinessLead
	pageToken := startPageToken
	pagesFetched := 0
	cycleFinished := false
	var lastNextToken string

	for pagesFetched < maxPages {
		req := newTextSearchRequest(query, pageSize, pageToken)
		apiResp, err := c.doTextSearch(ctx, req)

		// A fresh page token can take a moment to become valid on Google's side: wait and retry once.
		if err != nil && errors.Is(err, ErrInvalidPageToken) {
			if sleepErr := sleepCtx(ctx, c.pacing()); sleepErr != nil {
				return nil, "", pagesFetched, false, sleepErr
			}
			apiResp, err = c.doTextSearch(ctx, req)
		}

		if err != nil {
			// If earlier pages succeeded, treat an invalid/exhausted token as end-of-results.
			if len(allLeads) > 0 && errors.Is(err, ErrInvalidPageToken) {
				cycleFinished = true
				lastNextToken = ""
				break
			}
			return nil, "", pagesFetched, false, err
		}

		for _, raw := range apiResp.Places {
			allLeads = append(allLeads, ParseRawPlaceData(raw))
		}

		pagesFetched++
		lastNextToken = apiResp.NextPageToken

		if apiResp.NextPageToken == "" {
			cycleFinished = true
			break
		}

		pageToken = apiResp.NextPageToken

		if pagesFetched < maxPages {
			if err := sleepCtx(ctx, c.pacing()); err != nil {
				return nil, "", pagesFetched, false, err
			}
		}
	}

	return allLeads, lastNextToken, pagesFetched, cycleFinished, nil
}

func (c *PlacesAPIClient) pacing() time.Duration {
	if c.pagePacing <= 0 {
		return 2500 * time.Millisecond
	}
	return c.pagePacing
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
