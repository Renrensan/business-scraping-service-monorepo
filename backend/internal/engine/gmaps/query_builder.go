package gmaps

import (
	"strings"
)

type GeoBoundingBox struct {
	MinLat float64
	MaxLat float64
	MinLng float64
	MaxLng float64
}

func BuildSearchQuery(keyword, location, category string) string {
	var parts []string
	if keyword != "" {
		parts = append(parts, strings.TrimSpace(keyword))
	}
	if category != "" && !strings.EqualFold(keyword, category) {
		parts = append(parts, strings.TrimSpace(category))
	}
	if location != "" {
		parts = append(parts, "in "+strings.TrimSpace(location))
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
