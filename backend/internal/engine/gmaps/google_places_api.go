package gmaps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"business-scraping-service/backend/internal/model"
)

type PlacesAPIClient struct {
	apiKey     string
	httpClient *http.Client
}

func NewPlacesAPIClient(apiKey string) *PlacesAPIClient {
	return &PlacesAPIClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
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

	endpoint := "https://places.googleapis.com/v1/places:searchText"
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

		// Field Mask optimization (Single request extracts all Pro data for up to 20 places)
		fieldMask := "places.id,places.displayName,places.formattedAddress,places.location,places.types,places.primaryType,places.websiteUri,places.nationalPhoneNumber,places.rating,places.userRatingCount,places.businessStatus,places.googleMapsUri,places.regularOpeningHours,nextPageToken"

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
