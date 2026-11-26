package repository

// Listing leads: filters, sorting (including by scoring profile) and export.

import (
	"business-scraping-service/backend/internal/model"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// buildBusinessWhere turns a BusinessFilter into a SQL WHERE expression and its positional args.
func buildBusinessWhere(filter model.BusinessFilter) (string, []any) {
	whereClauses := []string{"1=1"}
	args := []any{}
	argIdx := 1

	if filter.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(name ILIKE $%d OR address ILIKE $%d OR city ILIKE $%d OR category ILIKE $%d OR subcategories::text ILIKE $%d OR website ILIKE $%d OR primary_phone ILIKE $%d OR emails::text ILIKE $%d)", argIdx, argIdx, argIdx, argIdx, argIdx, argIdx, argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}
	if filter.Category != "" {
		// Smart multi-keyword category matching (e.g. "Villas & Luxury Resorts" -> match Villa, Resort, etc.)
		cleanCat := strings.TrimSpace(filter.Category)
		terms := strings.FieldsFunc(cleanCat, func(r rune) bool {
			return r == '&' || r == '/' || r == ','
		})

		var catClauses []string
		for _, term := range terms {
			t := strings.TrimSpace(term)
			if len(t) > 2 {
				// Strip plural 's' for root match (e.g. "Villas" -> "Villa", "Cafes" -> "Cafe")
				root := strings.TrimSuffix(strings.TrimSuffix(t, "s"), "es")
				catClauses = append(catClauses, fmt.Sprintf("(category ILIKE $%d OR subcategories::text ILIKE $%d)", argIdx, argIdx))
				args = append(args, "%"+root+"%")
				argIdx++
			}
		}

		if len(catClauses) > 0 {
			whereClauses = append(whereClauses, "("+strings.Join(catClauses, " OR ")+")")
		} else {
			whereClauses = append(whereClauses, fmt.Sprintf("(category ILIKE $%d OR subcategories::text ILIKE $%d)", argIdx, argIdx))
			args = append(args, "%"+cleanCat+"%")
			argIdx++
		}
	}
	if filter.City != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("city ILIKE $%d", argIdx))
		args = append(args, "%"+filter.City+"%")
		argIdx++
	}
	if filter.State != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("state ILIKE $%d", argIdx))
		args = append(args, "%"+filter.State+"%")
		argIdx++
	}
	if filter.MinRating != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("rating >= $%d", argIdx))
		args = append(args, *filter.MinRating)
		argIdx++
	}
	if filter.Claimed != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("claimed = $%d", argIdx))
		args = append(args, *filter.Claimed)
		argIdx++
	}
	if filter.HasEmail != nil {
		if *filter.HasEmail {
			whereClauses = append(whereClauses, "(emails IS NOT NULL AND jsonb_typeof(emails) = 'array' AND jsonb_array_length(emails) > 0)")
		} else {
			whereClauses = append(whereClauses, "(emails IS NULL OR jsonb_typeof(emails) != 'array' OR jsonb_array_length(emails) = 0)")
		}
	}
	if filter.HasPhone != nil {
		if *filter.HasPhone {
			whereClauses = append(whereClauses, "((primary_phone IS NOT NULL AND primary_phone != '') OR (phones IS NOT NULL AND jsonb_typeof(phones) = 'array' AND jsonb_array_length(phones) > 0))")
		} else {
			whereClauses = append(whereClauses, "((primary_phone IS NULL OR primary_phone = '') AND (phones IS NULL OR jsonb_typeof(phones) != 'array' OR jsonb_array_length(phones) = 0))")
		}
	}
	if filter.HasWebsite != nil {
		if *filter.HasWebsite {
			whereClauses = append(whereClauses, "website != ''")
		} else {
			whereClauses = append(whereClauses, "website = ''")
		}
	}
	if filter.Contacted != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("contacted = $%d", argIdx))
		args = append(args, *filter.Contacted)
		argIdx++
	}
	if filter.JobID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("job_id = $%d", argIdx))
		args = append(args, *filter.JobID)
		argIdx++
	}
	if keywords := cleanStrings(filter.Keywords); len(keywords) > 0 {
		// A lead matches when ANY of the selected scrape keywords returned it.
		whereClauses = append(whereClauses, fmt.Sprintf("(scrape_keywords ?| $%d::text[] OR job_id IN (SELECT id FROM scrape_jobs WHERE query ILIKE ANY($%d::text[])))", argIdx, argIdx))
		args = append(args, keywords)
		argIdx++
	}
	if len(filter.Statuses) > 0 {
		statuses := make([]string, 0, len(filter.Statuses))
		for _, st := range filter.Statuses {
			if st.Valid() {
				statuses = append(statuses, string(st))
			}
		}
		if len(statuses) > 0 {
			whereClauses = append(whereClauses, fmt.Sprintf("lead_status = ANY($%d::text[])", argIdx))
			args = append(args, statuses)
			argIdx++
		}
	}
	if filter.OnlyNew != nil {
		maxReviews := 30
		if filter.MaxReviews != nil && *filter.MaxReviews > 0 {
			maxReviews = *filter.MaxReviews
		}
		if *filter.OnlyNew {
			whereClauses = append(whereClauses, fmt.Sprintf("(subcategories::text ILIKE '%%New Business%%' OR business_status = 'OPENING_SOON' OR review_count <= $%d)", argIdx))
			args = append(args, maxReviews)
			argIdx++
		} else {
			whereClauses = append(whereClauses, fmt.Sprintf("(subcategories::text NOT ILIKE '%%New Business%%' AND business_status != 'OPENING_SOON' AND review_count > $%d)", argIdx))
			args = append(args, maxReviews)
			argIdx++
		}
	} else if filter.MaxReviews != nil && *filter.MaxReviews > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("review_count <= $%d", argIdx))
		args = append(args, *filter.MaxReviews)
		argIdx++
	}

	_ = argIdx
	return strings.Join(whereClauses, " AND "), args
}

// cleanStrings trims entries and drops blanks and duplicates.
func cleanStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// List returns one page of leads for the UI. Page size is capped at 100.
func (r *BusinessRepository) List(ctx context.Context, filter model.BusinessFilter) (*model.BusinessListResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}
	return r.list(ctx, filter)
}

// list runs the filtered query with filter.Page/PageSize as given (no clamping).
func (r *BusinessRepository) list(ctx context.Context, filter model.BusinessFilter) (*model.BusinessListResponse, error) {
	offset := (filter.Page - 1) * filter.PageSize

	if filter.ScoreProfileID == nil && strings.TrimSpace(filter.ScoreProfileName) != "" {
		var id uuid.UUID
		err := r.db.QueryRow(ctx, `SELECT id FROM scoring_profiles WHERE LOWER(name) = LOWER($1)`, strings.TrimSpace(filter.ScoreProfileName)).Scan(&id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("scoring profile %q not found", filter.ScoreProfileName)
			}
			return nil, err
		}
		filter.ScoreProfileID = &id
	}

	whereSQL, args := buildBusinessWhere(filter)
	argIdx := len(args) + 1

	// With a scoring profile, join its scores (leads it doesn't apply to have none).
	fromSQL := "business_leads"
	scoreCols := ""
	scored := filter.ScoreProfileID != nil
	if scored {
		fromSQL = fmt.Sprintf("business_leads LEFT JOIN lead_scores ls ON ls.lead_id = business_leads.id AND ls.profile_id = $%d", argIdx)
		args = append(args, *filter.ScoreProfileID)
		argIdx++
		scoreCols = `, ls.score, ls.raw_score, ls.tier, ls.computed_tier, ls.tier_override, ls.note, ls.area_name, ls.breakdown, ls.computed_at`

		var tiers []string
		for _, t := range filter.Tiers {
			if t = strings.ToLower(strings.TrimSpace(t)); t != "" {
				tiers = append(tiers, t)
			}
		}
		if len(tiers) > 0 {
			whereSQL += fmt.Sprintf(" AND ls.tier = ANY($%d::text[])", argIdx)
			args = append(args, tiers)
			argIdx++
		}
		if area := strings.TrimSpace(filter.ScoreArea); area != "" {
			whereSQL += fmt.Sprintf(" AND LOWER(ls.area_name) = LOWER($%d)", argIdx)
			args = append(args, area)
			argIdx++
		}
	}

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", fromSQL, whereSQL)
	var totalCount int64
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, err
	}

	// Sorting
	sortBy := "created_at"
	switch strings.ToLower(filter.SortBy) {
	case "name":
		sortBy = "name"
	case "rating":
		sortBy = "rating"
	case "review_count":
		sortBy = "review_count"
	case "category":
		sortBy = "category"
	case "city":
		sortBy = "city"
	case "last_scraped_at":
		sortBy = "last_scraped_at"
	}

	sortOrder := "DESC"
	if strings.ToUpper(filter.SortOrder) == "ASC" {
		sortOrder = "ASC"
	}
	orderSQL := fmt.Sprintf("%s %s", sortBy, sortOrder)
	if scored && strings.EqualFold(filter.SortBy, "score") {
		if sortOrder == "ASC" {
			orderSQL = "ls.score ASC NULLS LAST, name"
		} else {
			// Best first by tier, then score; sister properties (same group) follow their group's best
			// property so they are pitched together. Unscored leads go last.
			orderSQL = `(ls.tier IS NULL),
				MIN(CASE ls.tier WHEN 'hot' THEN 0 WHEN 'good' THEN 1 WHEN 'low' THEN 2 WHEN 'skip' THEN 3 END) OVER grp,
				MAX(ls.score) OVER grp DESC,
				COALESCE(NULLIF(COALESCE(business_leads.group_name, business_leads.suggested_group_name), ''), business_leads.id::text),
				ls.score DESC, name`
		}
	}
	windowSQL := ""
	if scored {
		windowSQL = `WINDOW grp AS (PARTITION BY COALESCE(NULLIF(COALESCE(business_leads.group_name, business_leads.suggested_group_name), ''), business_leads.id::text))`
	}

	selectQuery := fmt.Sprintf(`
		SELECT id, job_id, place_id, cid, name, category, subcategories, address, street,
		       city, state, postal_code, country, latitude, longitude, rating, review_count,
		       price_level, claimed, business_status, website, primary_phone, emails, phones,
		       facebook_url, instagram_url, linkedin_url, twitter_url, youtube_url, tiktok_url,
		       google_maps_url, opening_hours, photos, enrichment_status, last_scraped_at,
		       last_enriched_at, contacted, contacted_at, lead_status, created_at, updated_at, scrape_keywords,
		       group_type, group_name, suggested_group_type, suggested_group_name%s
		FROM %s
		WHERE %s
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, scoreCols, fromSQL, whereSQL, windowSQL, orderSQL, argIdx, argIdx+1)

	queryArgs := append(args, filter.PageSize, offset)
	rows, err := r.db.Query(ctx, selectQuery, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*model.BusinessLead
	for rows.Next() {
		var lead model.BusinessLead
		var subcatBytes, emailsBytes, phonesBytes, hoursBytes, photosBytes, scrapeKwBytes []byte

		dest := []any{
			&lead.ID, &lead.JobID, &lead.PlaceID, &lead.CID, &lead.Name, &lead.Category, &subcatBytes, &lead.Address, &lead.Street,
			&lead.City, &lead.State, &lead.PostalCode, &lead.Country, &lead.Latitude, &lead.Longitude, &lead.Rating, &lead.ReviewCount,
			&lead.PriceLevel, &lead.Claimed, &lead.BusinessStatus, &lead.Website, &lead.PrimaryPhone, &emailsBytes, &phonesBytes,
			&lead.FacebookURL, &lead.InstagramURL, &lead.LinkedinURL, &lead.TwitterURL, &lead.YoutubeURL, &lead.TiktokURL,
			&lead.GoogleMapsURL, &hoursBytes, &photosBytes, &lead.EnrichmentStatus, &lead.LastScrapedAt,
			&lead.LastEnrichedAt, &lead.Contacted, &lead.ContactedAt, &lead.LeadStatus, &lead.CreatedAt, &lead.UpdatedAt, &scrapeKwBytes,
			&lead.GroupType, &lead.GroupName, &lead.SuggestedGroupType, &lead.SuggestedGroupName,
		}
		var sc scoreScan
		if scored {
			dest = append(dest, sc.dest()...)
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		if scored {
			lead.Score = sc.toScore(*filter.ScoreProfileID)
		}

		decodeForDisplay(subcatBytes, &lead.Subcategories, "business_leads", lead.ID, "subcategories")
		decodeForDisplay(scrapeKwBytes, &lead.ScrapeKeywords, "business_leads", lead.ID, "scrape_keywords")
		decodeForDisplay(emailsBytes, &lead.Emails, "business_leads", lead.ID, "emails")
		decodeForDisplay(phonesBytes, &lead.Phones, "business_leads", lead.ID, "phones")
		decodeForDisplay(hoursBytes, &lead.OpeningHours, "business_leads", lead.ID, "opening_hours")
		decodeForDisplay(photosBytes, &lead.Photos, "business_leads", lead.ID, "photos")

		items = append(items, &lead)
	}

	totalPages := int((totalCount + int64(filter.PageSize) - 1) / int64(filter.PageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	return &model.BusinessListResponse{
		Items:      items,
		TotalCount: totalCount,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

// exportMaxRows is the safety cap on rows returned by a single export.
const exportMaxRows = 10000

// GetAllForExport returns every lead matching the filter (all pages), up to exportMaxRows.
func (r *BusinessRepository) GetAllForExport(ctx context.Context, filter model.BusinessFilter) ([]*model.BusinessLead, error) {
	filter.Page = 1
	filter.PageSize = exportMaxRows
	// Call list directly: List clamps the page size to 100, which would cut exports short.
	resp, err := r.list(ctx, filter)
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}

// scoreScan receives the nullable lead_scores columns of a LEFT JOIN.
type scoreScan struct {
	score        *int
	raw          *float64
	tier         *string
	computedTier *string
	override     *string
	note         *string
	area         *string
	breakdown    []byte
	computedAt   *time.Time
}

func (s *scoreScan) dest() []any {
	return []any{&s.score, &s.raw, &s.tier, &s.computedTier, &s.override, &s.note, &s.area, &s.breakdown, &s.computedAt}
}

func (s *scoreScan) toScore(profileID uuid.UUID) *model.LeadScore {
	if s.score == nil {
		return nil
	}
	out := &model.LeadScore{ProfileID: profileID, Score: *s.score, TierOverride: s.override, Breakdown: s.breakdown}
	if s.raw != nil {
		out.RawScore = *s.raw
	}
	if s.tier != nil {
		out.Tier = *s.tier
	}
	if s.computedTier != nil {
		out.ComputedTier = *s.computedTier
	}
	if s.note != nil {
		out.Note = *s.note
	}
	if s.area != nil {
		out.AreaName = *s.area
	}
	if s.computedAt != nil {
		out.ComputedAt = *s.computedAt
	}
	return out
}
