package enrichment

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	emailRegex  = regexp.MustCompile(`(?i)[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	junkDomains = map[string]bool{
		"example.com":    true,
		"domain.com":     true,
		"email.com":      true,
		"wixpress.com":   true,
		"sentry.io":      true,
		"cloudflare.com": true,
		"schema.org":     true,
		"google.com":     true,
		"googleapis.com": true,
		"w3.org":         true,
		"bootstrap.com":  true,
		"wordpress.org":  true,
		"wordpress.com":  true,
		"gravatar.com":   true,
		"yourdomain.com": true,
		"mycompany.com":  true,
	}
	junkExtensions = []string{".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".css", ".js"}
)

func ExtractEmails(htmlContent string) []string {
	found := make(map[string]bool)
	var emails []string

	// 1. Direct regex
	matches := emailRegex.FindAllString(htmlContent, -1)
	for _, raw := range matches {
		cleaned := cleanEmail(raw)
		if isValidEmail(cleaned) && !found[cleaned] {
			found[cleaned] = true
			emails = append(emails, cleaned)
		}
	}

	// 2. mailto: extraction
	mailtoRegex := regexp.MustCompile(`(?i)href\s*=\s*["']mailto:([^"'\?>]+)`)
	mailtoMatches := mailtoRegex.FindAllStringSubmatch(htmlContent, -1)
	for _, m := range mailtoMatches {
		if len(m) > 1 {
			raw := strings.TrimSpace(m[1])
			if unescaped, err := url.QueryUnescape(raw); err == nil {
				raw = unescaped
			}
			cleaned := cleanEmail(raw)
			if isValidEmail(cleaned) && !found[cleaned] {
				found[cleaned] = true
				emails = append(emails, cleaned)
			}
		}
	}

	return emails
}

func cleanEmail(e string) string {
	e = strings.TrimSpace(strings.ToLower(e))
	e = strings.Trim(e, ".,;:()'\"<>[]{}")
	return e
}

func isValidEmail(e string) bool {
	if len(e) < 5 || len(e) > 100 {
		return false
	}
	parts := strings.Split(e, "@")
	if len(parts) != 2 {
		return false
	}
	domain := parts[1]
	if junkDomains[domain] {
		return false
	}
	for _, ext := range junkExtensions {
		if strings.HasSuffix(domain, ext) {
			return false
		}
	}
	return true
}
