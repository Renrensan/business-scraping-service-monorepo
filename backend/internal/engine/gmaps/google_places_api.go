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

var ErrInvalidPageToken = errors.New("invalid or expired page token")

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

type PlacesAPIClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	pagePacing time.Duration
}

func NewPlacesAPIClient(apiKey string) *PlacesAPIClient {
	return &PlacesAPIClient{
		apiKey:     apiKey,
		baseURL:    "https://places.googleapis.com",
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		pagePacing: 2500 * time.Millisecond,
	}
}

type TextSearchRequest struct {
	TextQuery    string `json:"textQuery"`
	IncludedType string `json:"includedType,omitempty"`
	PageSize     int    `json:"pageSize,omitempty"`
	PageToken    string `json:"pageToken,omitempty"`
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

// SearchPlacesText executes a Text Search (New) with optimized Pro-tier field mask
func (c *PlacesAPIClient) SearchPlacesText(ctx context.Context, query string, maxResults int) ([]*model.BusinessLead, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("google api key is not configured")
	}

	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://places.googleapis.com"
	}
	endpoint := baseURL + "/v1/places:searchText"
	pageSize := 20
	if maxResults > 0 && maxResults < 20 {
		pageSize = maxResults
	}

	var allLeads []*model.BusinessLead
	pageToken := ""

	for {
		reqBody := TextSearchRequest{
			TextQuery: query,
			PageSize:  pageSize,
			PageToken: pageToken,
		}

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return nil, err
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, err
		}

		// Field Mask optimization (Single request extracts all Pro data + recency signals for up to 20 places)
		fieldMask := "places.id,places.displayName,places.formattedAddress,places.location,places.types,places.primaryType,places.websiteUri,places.nationalPhoneNumber,places.rating,places.userRatingCount,places.businessStatus,places.googleMapsUri,places.regularOpeningHours,places.editorialSummary,places.reviews,nextPageToken"

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-Goog-Api-Key", c.apiKey)
		httpReq.Header.Set("X-Goog-FieldMask", fieldMask)

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("failed to call places api: %w", err)
		}
		defer resp.Body.Close()

		respBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var apiResp TextSearchResponse
		if err := json.Unmarshal(respBytes, &apiResp); err != nil {
			return nil, fmt.Errorf("failed to decode places api response: %w", err)
		}

		if apiResp.Error != nil {
			return nil, fmt.Errorf("google places api error [%s]: %s", apiResp.Error.Status, apiResp.Error.Message)
		}

		for _, raw := range apiResp.Places {
			lead := ParseRawPlaceData(raw)
			allLeads = append(allLeads, lead)
			if maxResults > 0 && len(allLeads) >= maxResults {
				return allLeads, nil
			}
		}

		if apiResp.NextPageToken == "" || (maxResults > 0 && len(allLeads) >= maxResults) {
			break
		}

		pageToken = apiResp.NextPageToken
		time.Sleep(1 * time.Second) // rate pace
	}

	return allLeads, nil
}

// SearchPlacesTextWithCursor executes a paginated Text Search starting from a given pageToken, traversing up to maxPages
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

	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://places.googleapis.com"
	}
	endpoint := baseURL + "/v1/places:searchText"
	var allLeads []*model.BusinessLead
	pageToken := startPageToken
	pagesFetched := 0
	cycleFinished := false
	var lastNextToken string

	pacing := c.pagePacing
	if pacing <= 0 {
		pacing = 2500 * time.Millisecond
	}

	for pagesFetched < maxPages {
		reqBody := TextSearchRequest{
			TextQuery: query,
			PageSize:  pageSize,
			PageToken: pageToken,
		}

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return nil, "", pagesFetched, false, err
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, "", pagesFetched, false, err
		}

		fieldMask := "places.id,places.displayName,places.formattedAddress,places.location,places.types,places.primaryType,places.websiteUri,places.nationalPhoneNumber,places.rating,places.userRatingCount,places.businessStatus,places.googleMapsUri,places.regularOpeningHours,places.editorialSummary,places.reviews,nextPageToken"

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-Goog-Api-Key", c.apiKey)
		httpReq.Header.Set("X-Goog-FieldMask", fieldMask)

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			return nil, "", pagesFetched, false, fmt.Errorf("failed to call places api: %w", err)
		}

		respBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, "", pagesFetched, false, err
		}

		var apiResp TextSearchResponse
		if err := json.Unmarshal(respBytes, &apiResp); err != nil {
			return nil, "", pagesFetched, false, fmt.Errorf("failed to decode places api response: %w", err)
		}

		// If Google rejects page token, sleep pacing and retry once (for Google token propagation delay)
		if apiResp.Error != nil && pageToken != "" {
			time.Sleep(pacing)
			if retryReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonBody)); err == nil {
				retryReq.Header.Set("Content-Type", "application/json")
				retryReq.Header.Set("X-Goog-Api-Key", c.apiKey)
				retryReq.Header.Set("X-Goog-FieldMask", fieldMask)
				if retryResp, err := c.httpClient.Do(retryReq); err == nil {
					if retryBytes, err := io.ReadAll(retryResp.Body); err == nil {
						var retryAPIResp TextSearchResponse
						if err := json.Unmarshal(retryBytes, &retryAPIResp); err == nil {
							apiResp = retryAPIResp
						}
					}
					retryResp.Body.Close()
				}
			}
		}

		if apiResp.Error != nil {
			errStatus := apiResp.Error.Status
			errMsg := apiResp.Error.Message
			isTokenErr := pageToken != "" && (errStatus == "INVALID_ARGUMENT" || strings.Contains(strings.ToLower(errMsg), "pagetoken") || strings.Contains(strings.ToLower(errMsg), "paging"))

			// If earlier pages succeeded, gracefully treat invalid/exhausted token as end-of-results
			if len(allLeads) > 0 && isTokenErr {
				cycleFinished = true
				lastNextToken = ""
				break
			}

			if isTokenErr {
				return nil, "", pagesFetched, false, fmt.Errorf("%w: google places api [%s]: %s", ErrInvalidPageToken, errStatus, errMsg)
			}
			return nil, "", pagesFetched, false, fmt.Errorf("google places api error [%s]: %s", errStatus, errMsg)
		}

		for _, raw := range apiResp.Places {
			lead := ParseRawPlaceData(raw)
			allLeads = append(allLeads, lead)
		}

		pagesFetched++
		lastNextToken = apiResp.NextPageToken

		if apiResp.NextPageToken == "" {
			cycleFinished = true
			break
		}

		pageToken = apiResp.NextPageToken

		if pagesFetched < maxPages {
			time.Sleep(pacing)
		}
	}

	return allLeads, lastNextToken, pagesFetched, cycleFinished, nil
}
