package enrichment

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"business-scraping-service/backend/internal/engine/proxy"
)

type EnrichmentResult struct {
	Emails  []string
	Phones  []string
	Socials SocialLinks
}

type Crawler struct {
	rotator   *proxy.Rotator
	userAgent string
}

func NewCrawler(rotator *proxy.Rotator, defaultUserAgent string) *Crawler {
	if rotator == nil {
		rotator = proxy.NewRotator(nil)
	}
	if defaultUserAgent == "" {
		defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"
	}
	return &Crawler{
		rotator:   rotator,
		userAgent: defaultUserAgent,
	}
}

func (c *Crawler) EnrichWebsite(ctx context.Context, websiteURL string) (*EnrichmentResult, error) {
	if websiteURL == "" {
		return nil, fmt.Errorf("empty website url")
	}

	if !strings.HasPrefix(websiteURL, "http://") && !strings.HasPrefix(websiteURL, "https://") {
		websiteURL = "https://" + websiteURL
	}

	parsedURL, err := url.Parse(websiteURL)
	if err != nil {
		return nil, err
	}

	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)
	candidatePaths := []string{
		parsedURL.RequestURI(),
		"/contact",
		"/contact-us",
		"/about",
		"/about-us",
	}

	visited := make(map[string]bool)
	var combinedHTML strings.Builder

	var client *http.Client
	if c.rotator != nil {
		client = c.rotator.GetNextClient(6*time.Second, c.userAgent)
	} else {
		client = proxy.NewRotator(nil).GetNextClient(6*time.Second, c.userAgent)
	}

	for _, path := range candidatePaths {
		targetURL := baseURL + path
		if visited[targetURL] {
			continue
		}
		visited[targetURL] = true

		html, err := c.fetchPage(ctx, client, targetURL)
		if err == nil && len(html) > 0 {
			combinedHTML.WriteString(html)
			combinedHTML.WriteString("\n")
		}
	}

	fullContent := combinedHTML.String()
	emails := ExtractEmails(fullContent)
	socials := ExtractSocials(fullContent)

	return &EnrichmentResult{
		Emails:  emails,
		Phones:  []string{},
		Socials: socials,
	}, nil
}

func (c *Crawler) fetchPage(ctx context.Context, client *http.Client, pageURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return "", fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	limitReader := io.LimitReader(resp.Body, 2*1024*1024)
	bytes, err := io.ReadAll(limitReader)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}
