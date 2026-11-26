package gmaps

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"business-scraping-service/backend/internal/model"
)

var (
	sanitizerRegex = regexp.MustCompile(`[^a-zA-Z0-9]+`)
)

func sanitizeKey(s string) string {
	s = strings.ToLower(s)
	return sanitizerRegex.ReplaceAllString(s, "_")
}

func ParseRawPlaceData(data map[string]any) *model.BusinessLead {
	lead := &model.BusinessLead{
		EnrichmentStatus: model.EnrichmentNone,
		BusinessStatus:   "OPERATIONAL",
		Claimed:          true,
	}

	if id, ok := data["id"].(string); ok {
		lead.PlaceID = strings.TrimPrefix(id, "places/")
	}
	if name, ok := data["name"].(string); ok {
		lead.Name = name
	}
	if displayName, ok := data["displayName"].(map[string]any); ok {
		if text, ok := displayName["text"].(string); ok {
			lead.Name = text
		}
	}
	if category, ok := data["primaryType"].(string); ok {
		lead.Category = formatCategory(category)
	} else if category, ok := data["category"].(string); ok {
		lead.Category = category
	}
	if types, ok := data["types"].([]any); ok {
		for _, t := range types {
			if s, ok := t.(string); ok {
				lead.Subcategories = append(lead.Subcategories, formatCategory(s))
			}
		}
	}
	if address, ok := data["formattedAddress"].(string); ok {
		lead.Address = address
		parseAddressParts(lead, address)
	} else if address, ok := data["address"].(string); ok {
		lead.Address = address
		parseAddressParts(lead, address)
	}
	if location, ok := data["location"].(map[string]any); ok {
		if lat, ok := location["latitude"].(float64); ok {
			lead.Latitude = lat
		}
		if lng, ok := location["longitude"].(float64); ok {
			lead.Longitude = lng
		}
	}
	if rating, ok := data["rating"].(float64); ok {
		lead.Rating = rating
	}
	if reviewCount, ok := data["userRatingCount"].(float64); ok {
		lead.ReviewCount = int(reviewCount)
	} else if reviewCount, ok := data["reviewCount"].(int); ok {
		lead.ReviewCount = reviewCount
	}
	if website, ok := data["websiteUri"].(string); ok {
		lead.Website = website
	} else if website, ok := data["website"].(string); ok {
		lead.Website = website
	}
	if phone, ok := data["nationalPhoneNumber"].(string); ok {
		lead.PrimaryPhone = phone
	} else if phone, ok := data["phone"].(string); ok {
		lead.PrimaryPhone = phone
	}
	if priceLevel, ok := data["priceLevel"].(string); ok {
		lead.PriceLevel = formatPriceLevel(priceLevel)
	}
	if status, ok := data["businessStatus"].(string); ok {
		lead.BusinessStatus = status
	}
	if mapsURI, ok := data["googleMapsUri"].(string); ok {
		lead.GoogleMapsURL = mapsURI
	}

	if lead.PlaceID == "" && lead.Name != "" {
		lead.PlaceID = fmt.Sprintf("place_%s_%f_%f", sanitizeKey(lead.Name), lead.Latitude, lead.Longitude)
	}

	// Evaluate Google native signals for newly opened / uploaded status
	_, isNew := EvaluateNewBusiness(lead, data, false)
	if isNew {
		lead.Subcategories = append(lead.Subcategories, "New Business")
	}

	return lead
}

// EvaluateNewBusiness calculates a recency confidence score based directly on Google metadata
func EvaluateNewBusiness(lead *model.BusinessLead, rawData map[string]any, matchedNewQuery bool) (int, bool) {
	score := 0

	// 1. Google Business Status (OPENING_SOON = 100% pre-launch / grand opening)
	if strings.EqualFold(lead.BusinessStatus, "OPENING_SOON") {
		score += 100
		return score, true
	}

	// 2. Google Editorial Summary / Description Launch Indicators
	var summaryText string
	if ed, ok := rawData["editorialSummary"].(map[string]any); ok {
		if txt, ok := ed["text"].(string); ok {
			summaryText = txt
		}
	} else if desc, ok := rawData["description"].(string); ok {
		summaryText = desc
	}

	if summaryText != "" {
		lowerSum := strings.ToLower(summaryText)
		launchKeywords := []string{
			"newly opened", "grand opening", "soft opening", "just opened",
			"opened in 202", "opened this", "new location", "brand new",
			"coming soon", "now open",
		}
		for _, kw := range launchKeywords {
			if strings.Contains(lowerSum, kw) {
				score += 50
				break
			}
		}
	}

	// 3. Earliest Review Timestamps / Relative Date Analysis
	if reviews, ok := rawData["reviews"].([]any); ok && len(reviews) > 0 {
		hasOldReview := false
		allRecent := true
		now := time.Now().UTC()

		for _, r := range reviews {
			revMap, ok := r.(map[string]any)
			if !ok {
				continue
			}

			// Check RFC3339 publishTime
			if ptStr, ok := revMap["publishTime"].(string); ok && ptStr != "" {
				if pt, err := time.Parse(time.RFC3339, ptStr); err == nil {
					ageDays := now.Sub(pt).Hours() / 24
					if ageDays > 180 {
						hasOldReview = true
						allRecent = false
					} else if ageDays > 90 {
						allRecent = false
					}
				}
			}

			// Check relative publish time description (e.g. "a year ago", "3 years ago")
			if rel, ok := revMap["relativePublishTimeDescription"].(string); ok {
				lowerRel := strings.ToLower(rel)
				if strings.Contains(lowerRel, "year") ||
					(strings.Contains(lowerRel, "month") && (strings.Contains(lowerRel, "4") || strings.Contains(lowerRel, "5") || strings.Contains(lowerRel, "6") || strings.Contains(lowerRel, "7") || strings.Contains(lowerRel, "8") || strings.Contains(lowerRel, "9") || strings.Contains(lowerRel, "10") || strings.Contains(lowerRel, "11"))) {
					hasOldReview = true
					allRecent = false
				}
			}
		}

		if !hasOldReview {
			if allRecent {
				score += 70
			} else {
				score += 30
			}
		}
	} else if lead.ReviewCount == 0 {
		// Zero reviews on a verified listing indicates a newly minted Google Maps pin
		score += 40
	}

	// 4. Query modifier match from Google's recent index
	if matchedNewQuery {
		score += 30
	}

	isNew := score >= 50
	return score, isNew
}

func parseAddressParts(lead *model.BusinessLead, address string) {
	parts := strings.Split(address, ",")
	if len(parts) >= 3 {
		lead.Street = strings.TrimSpace(parts[0])
		lead.City = strings.TrimSpace(parts[1])
		stateZip := strings.TrimSpace(parts[2])
		szParts := strings.Fields(stateZip)
		if len(szParts) >= 1 {
			lead.State = szParts[0]
		}
		if len(szParts) >= 2 {
			lead.PostalCode = szParts[1]
		}
		if len(parts) >= 4 {
			lead.Country = strings.TrimSpace(parts[3])
		} else {
			lead.Country = "USA"
		}
	}
}

// formatCategory turns a Places type ("resort_hotel") into a label ("Resort Hotel").
// Places types are ASCII snake_case, so a simple per-word capitalisation is enough
// (strings.Title is deprecated).
func formatCategory(raw string) string {
	words := strings.Fields(strings.ReplaceAll(raw, "_", " "))
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

func formatPriceLevel(p string) string {
	switch p {
	case "PRICE_LEVEL_INEXPENSIVE":
		return "$"
	case "PRICE_LEVEL_MODERATE":
		return "$$"
	case "PRICE_LEVEL_EXPENSIVE":
		return "$$$"
	case "PRICE_LEVEL_VERY_EXPENSIVE":
		return "$$$$"
	default:
		return p
	}
}
