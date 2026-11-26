package repository

// Dashboard metrics, scrape keywords and derived categories for leads.

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"business-scraping-service/backend/internal/model"
)

func (r *BusinessRepository) GetOverviewMetrics(ctx context.Context) (*model.OverviewMetrics, error) {
	var metrics model.OverviewMetrics

	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE emails IS NOT NULL AND jsonb_typeof(emails) = 'array' AND jsonb_array_length(emails) > 0),
		       COUNT(*) FILTER (WHERE (primary_phone IS NOT NULL AND primary_phone != '')
		                           OR (phones IS NOT NULL AND jsonb_typeof(phones) = 'array' AND jsonb_array_length(phones) > 0)),
		       COUNT(*) FILTER (WHERE enrichment_status = 'SUCCESS')
		FROM business_leads`).Scan(&metrics.TotalLeads, &metrics.LeadsWithEmail, &metrics.LeadsWithPhone, &metrics.EnrichedLeads)
	if err != nil {
		return nil, fmt.Errorf("counting leads: %w", err)
	}

	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE status = 'PROCESSING'),
		       COUNT(*) FILTER (WHERE status = 'COMPLETED'),
		       COALESCE(SUM(total_scraped), 0)
		FROM scrape_jobs`).Scan(&metrics.ActiveJobs, &metrics.CompletedJobs, &metrics.TotalScrapedEver)
	if err != nil {
		return nil, fmt.Errorf("counting jobs: %w", err)
	}

	if metrics.TotalLeads > 0 {
		metrics.EmailHitRate = (float64(metrics.LeadsWithEmail) / float64(metrics.TotalLeads)) * 100
		metrics.PhoneHitRate = (float64(metrics.LeadsWithPhone) / float64(metrics.TotalLeads)) * 100
	}

	return &metrics, nil
}

func (r *BusinessRepository) GetCategoryStats(ctx context.Context, limit int) ([]*model.CategoryStat, error) {
	if limit <= 0 {
		limit = 10
	}
	query := `
		SELECT category, COUNT(*) as cnt
		FROM business_leads
		WHERE category != ''
		GROUP BY category
		ORDER BY cnt DESC
		LIMIT $1
	`
	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*model.CategoryStat
	for rows.Next() {
		var stat model.CategoryStat
		if err := rows.Scan(&stat.Category, &stat.Count); err != nil {
			return nil, err
		}
		stats = append(stats, &stat)
	}
	return stats, rows.Err()
}

func (r *BusinessRepository) GetLocationStats(ctx context.Context, limit int) ([]*model.LocationStat, error) {
	if limit <= 0 {
		limit = 10
	}
	query := `
		SELECT city, state, COUNT(*) as cnt
		FROM business_leads
		WHERE city != ''
		GROUP BY city, state
		ORDER BY cnt DESC
		LIMIT $1
	`
	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*model.LocationStat
	for rows.Next() {
		var stat model.LocationStat
		if err := rows.Scan(&stat.City, &stat.State, &stat.Count); err != nil {
			return nil, err
		}
		stats = append(stats, &stat)
	}
	return stats, rows.Err()
}

func (r *BusinessRepository) GetScrapeKeywords(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT keyword FROM (
			SELECT keyword FROM scraper_keywords WHERE keyword != ''
			UNION
			SELECT query AS keyword FROM scrape_jobs WHERE query != ''
		) combined_keywords
		ORDER BY keyword ASC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keywords []string
	for rows.Next() {
		var kw string
		if err := rows.Scan(&kw); err == nil && kw != "" {
			keywords = append(keywords, kw)
		}
	}
	return keywords, rows.Err()
}

var locationSepRegex = regexp.MustCompile(`(?i)\s+(in|at|near|around|di)\s+|,|-|\|`)

// DeriveCategoryFromKeyword dynamically extracts and title-cases the front category word(s)
// from a scrape keyword (e.g. "cafe in canggu" -> "Cafe", "cafe in denpasar" -> "Cafe").
func DeriveCategoryFromKeyword(kw string) string {
	clean := strings.TrimSpace(kw)
	if clean == "" {
		return ""
	}

	// Split by location separators
	loc := locationSepRegex.FindStringIndex(clean)
	if len(loc) > 0 {
		clean = strings.TrimSpace(clean[:loc[0]])
	}

	if clean == "" {
		return ""
	}

	// Title Case words
	words := strings.Fields(clean)
	for i, w := range words {
		lower := strings.ToLower(w)
		if len(lower) > 0 {
			words[i] = strings.ToUpper(lower[:1]) + lower[1:]
		}
	}
	derived := strings.Join(words, " ")

	// Normalize trailing plural 's' if > 4 chars (e.g. "Cafes" -> "Cafe", "Villas" -> "Villa")
	// but preserve words ending in 'ss' like "Fitness" or "Wellness"
	if strings.HasSuffix(derived, "s") && !strings.HasSuffix(derived, "ss") && len(derived) > 4 {
		derived = strings.TrimSuffix(derived, "s")
	}

	return derived
}

// GetDerivedCategories lists the real Google categories present in the data, for the Category filter.
// (It used to derive them from scrape keyword names, but those never match Google's own naming —
// "petshop" vs "Pet Store" — and pushed filtering onto the keyword tags instead.)
func (r *BusinessRepository) GetDerivedCategories(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT category FROM business_leads
		WHERE TRIM(COALESCE(category, '')) != ''
		GROUP BY category
		ORDER BY category ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := []string{}
	for rows.Next() {
		var cat string
		if err := rows.Scan(&cat); err == nil && cat != "" {
			categories = append(categories, cat)
		}
	}
	return categories, rows.Err()
}
