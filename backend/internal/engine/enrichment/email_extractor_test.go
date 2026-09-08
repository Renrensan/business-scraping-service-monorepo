package enrichment

import (
	"testing"
)

func TestExtractEmails(t *testing.T) {
	html := `
		<html>
			<body>
				<p>Contact us at info@mybusiness.com or sales@mybusiness.com.</p>
				<a href="mailto:support@mybusiness.com?subject=Help">Support</a>
				<p>Ignore junk: dev@example.com, test@sentry.io, image.png@example.com</p>
			</body>
		</html>
	`
	emails := ExtractEmails(html)

	expected := map[string]bool{
		"info@mybusiness.com":    true,
		"sales@mybusiness.com":   true,
		"support@mybusiness.com": true,
	}

	for _, e := range emails {
		if !expected[e] {
			t.Errorf("Unexpected email found: %s", e)
		}
	}

	if len(emails) != 3 {
		t.Errorf("Expected 3 emails, got %d: %v", len(emails), emails)
	}
}

func TestExtractSocials(t *testing.T) {
	html := `
		<div>
			<a href="https://www.facebook.com/mycoffee-shop">Facebook</a>
			<a href="https://instagram.com/mycoffeeshop">Instagram</a>
			<a href="https://linkedin.com/company/my-coffee-corp">LinkedIn</a>
			<a href="https://x.com/mycoffee_hq">X</a>
			<a href="https://www.youtube.com/@mycoffeeroasters">YouTube</a>
		</div>
	`
	socials := ExtractSocials(html)

	if socials.Facebook != "https://www.facebook.com/mycoffee-shop" {
		t.Errorf("Unexpected FB: %s", socials.Facebook)
	}
	if socials.Instagram != "https://instagram.com/mycoffeeshop" {
		t.Errorf("Unexpected IG: %s", socials.Instagram)
	}
	if socials.LinkedIn != "https://linkedin.com/company/my-coffee-corp" {
		t.Errorf("Unexpected LinkedIn: %s", socials.LinkedIn)
	}
	if socials.Twitter != "https://x.com/mycoffee_hq" {
		t.Errorf("Unexpected Twitter: %s", socials.Twitter)
	}
	if socials.YouTube != "https://www.youtube.com/@mycoffeeroasters" {
		t.Errorf("Unexpected YouTube: %s", socials.YouTube)
	}
}
