package utils

import "strings"

// ToWhatsAppNumber converts a phone number as stored from Google Maps (usually Indonesian
// national format, e.g. "0812-3456-7890") into international format ("+6281234567890").
//
// isMobile reports whether the number can plausibly have WhatsApp. Indonesian mobiles are
// +62 8[1-9]…; landlines (e.g. Bali 0361…) and toll-free 080x numbers are not mobile.
// Numbers from other countries (written with + or 00) can't be classified, so they count as mobile.
//
// ok is false when the input cannot be turned into a usable number.
func ToWhatsAppNumber(raw string) (number string, isMobile bool, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false, false
	}

	hasPlus := strings.HasPrefix(raw, "+")
	var sb strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}
	d := sb.String()

	switch {
	case hasPlus:
		// already international
	case strings.HasPrefix(d, "00"):
		d = d[2:]
	case strings.HasPrefix(d, "62"):
		// international without the +
	case strings.HasPrefix(d, "0"):
		d = "62" + d[1:]
	case strings.HasPrefix(d, "8"):
		d = "62" + d
	default:
		return "", false, false
	}

	// E.164 numbers are at most 15 digits; anything under 10 is too short to be real here.
	if len(d) < 10 || len(d) > 15 {
		return "", false, false
	}

	if strings.HasPrefix(d, "62") {
		isMobile = len(d) >= 11 && d[2] == '8' && d[3] >= '1' && d[3] <= '9'
	} else {
		isMobile = true
	}
	return "+" + d, isMobile, true
}
