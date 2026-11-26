package gmaps

import (
	"regexp"
	"strings"
)

// lodgingTerms are search words Google Places misreads without a type filter: "Villa in Jimbaran, Bali"
// returns nothing, probably because "Villa ..." is a common Indonesian street/housing-complex name and
// the query is read as an address. Restricting to the lodging type makes these queries return places.
// Keep this list to terms verified as broken without the filter; adding a type also narrows results.
var lodgingTerms = regexp.MustCompile(`(?i)\b(villas?|homestays?|guest ?houses?|bungalows?)\b`)

// InferIncludedType returns the Places API includedType to send with a query, or "" for none.
// It must be deterministic: page-token requests have to repeat the first request's parameters.
func InferIncludedType(query string) string {
	if lodgingTerms.MatchString(query) {
		return "lodging"
	}
	return ""
}

type GeoBoundingBox struct {
	MinLat float64
	MaxLat float64
	MinLng float64
	MaxLng float64
}

func BuildSearchQuery(keyword, location, category string) string {
	return BuildSearchQueryWithFilter(keyword, location, category, false)
}

func BuildSearchQueryWithFilter(keyword, location, category string, newOnly bool) string {
	var parts []string
	cleanKeyword := strings.TrimSpace(keyword)
	cleanLocation := strings.TrimSpace(location)
	cleanCategory := strings.TrimSpace(category)

	// If category is "All Categories", "Any Business", "All Businesses", ignore it
	if strings.EqualFold(cleanCategory, "all categories") ||
		strings.EqualFold(cleanCategory, "any business") ||
		strings.EqualFold(cleanCategory, "all businesses") ||
		strings.EqualFold(cleanCategory, "all") {
		cleanCategory = ""
	}

	if newOnly && !strings.Contains(strings.ToLower(cleanKeyword), "new") {
		parts = append(parts, "newly opened")
	}

	if cleanKeyword != "" {
		parts = append(parts, cleanKeyword)
	}
	if cleanCategory != "" && !strings.EqualFold(cleanKeyword, cleanCategory) {
		parts = append(parts, cleanCategory)
	}

	// If no specific keyword or category is requested, default to "businesses"
	if len(parts) == 0 || (len(parts) == 1 && parts[0] == "newly opened") {
		parts = append(parts, "businesses")
	}

	if cleanLocation != "" {
		parts = append(parts, "in "+cleanLocation)
	}
	return strings.Join(parts, " ")
}

// GenerateGridTiles creates sub-coordinate grid points to cover larger geographic areas
func GenerateGridTiles(centerLat, centerLng, radiusKm float64, steps int) [][2]float64 {
	var coords [][2]float64
	coords = append(coords, [2]float64{centerLat, centerLng})

	if steps <= 1 {
		return coords
	}

	latDelta := (radiusKm / 111.0) / float64(steps)
	lngDelta := (radiusKm / (111.0 * 0.7)) / float64(steps)

	for i := -steps; i <= steps; i++ {
		for j := -steps; j <= steps; j++ {
			if i == 0 && j == 0 {
				continue
			}
			lat := centerLat + float64(i)*latDelta
			lng := centerLng + float64(j)*lngDelta
			coords = append(coords, [2]float64{lat, lng})
		}
	}

	return coords
}
