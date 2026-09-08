package gmaps

import (
	"fmt"
	"regexp"
	"strings"

	"business-scraping-service/backend/internal/model"
)

var (
	sanitizerRegex = regexp.MustCompile(`[^a-zA-Z0-9]+`)
)

func sanitizeKey(s string) string {
	s = strings.ToLower(s)
	return sanitizerRegex.ReplaceAllString(s, "_")
}

func ParseRawPlaceData(data map[string]interface{}) *model.BusinessLead {
	lead := &model.BusinessLead{
		EnrichmentStatus: model.EnrichmentNone,
		BusinessStatus:   "OPERATIONAL",
		Claimed:          true,
	}

	if id, ok := data["id"].(string); ok {
		lead.PlaceID = id
	}
	if name, ok := data["name"].(string); ok {
		lead.Name = name
	}
	if displayName, ok := data["displayName"].(map[string]interface{}); ok {
		if text, ok := displayName["text"].(string); ok {
			lead.Name = text
		}
	}
	if category, ok := data["primaryType"].(string); ok {
		lead.Category = formatCategory(category)
	} else if category, ok := data["category"].(string); ok {
		lead.Category = category
	}
	if types, ok := data["types"].([]interface{}); ok {
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
	if location, ok := data["location"].(map[string]interface{}); ok {
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

	return lead
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

func formatCategory(raw string) string {
	raw = strings.ReplaceAll(raw, "_", " ")
	return strings.Title(raw)
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
