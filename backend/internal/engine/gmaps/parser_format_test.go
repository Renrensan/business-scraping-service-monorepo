package gmaps

import "testing"

func TestFormatCategory(t *testing.T) {
	// Same labels strings.Title produced for real Places types seen in production.
	tests := map[string]string{
		"lodging":            "Lodging",
		"resort_hotel":       "Resort Hotel",
		"private_guest_room": "Private Guest Room",
		"bed_and_breakfast":  "Bed And Breakfast",
		"point_of_interest":  "Point Of Interest",
		"spa":                "Spa",
		"veterinary_care":    "Veterinary Care",
		"Hotel":              "Hotel",
		"":                   "",
	}
	for in, want := range tests {
		if got := formatCategory(in); got != want {
			t.Errorf("formatCategory(%q) = %q, want %q", in, got, want)
		}
	}
}
