package repository

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"business-scraping-service/backend/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BusinessRepository struct {
	db *pgxpool.Pool
}

func NewBusinessRepository(db *pgxpool.Pool) *BusinessRepository {
	return &BusinessRepository{db: db}
}

func safeJSONArray(items []string) []byte {
	if len(items) == 0 {
		return []byte("[]")
	}
	b, err := json.Marshal(items)
	if err != nil || string(b) == "null" {
		return []byte("[]")
	}
	return b
}

func safeJSONObject(obj map[string]interface{}) []byte {
	if len(obj) == 0 {
		return []byte("{}")
	}
	b, err := json.Marshal(obj)
	if err != nil || string(b) == "null" {
		return []byte("{}")
	}
	return b
}

func normalizeDomain(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return ""
	}
	raw = strings.TrimPrefix(raw, "https://")
	raw = strings.TrimPrefix(raw, "http://")
	raw = strings.TrimPrefix(raw, "www.")
	if idx := strings.IndexAny(raw, "/?#"); idx != -1 {
		raw = raw[:idx]
	}
	return strings.TrimSpace(raw)
}

// sharedWebsiteHosts are domains many unrelated businesses list as their "website";
// matching on them would merge different businesses into one row.
var sharedWebsiteHosts = map[string]bool{
	"facebook.com": true, "m.facebook.com": true, "fb.com": true, "fb.me": true,
	"instagram.com": true, "tiktok.com": true, "twitter.com": true, "x.com": true,
	"linkedin.com": true, "youtube.com": true, "linktr.ee": true, "wa.me": true,
	"api.whatsapp.com": true, "whatsapp.com": true, "bit.ly": true, "google.com": true,
	"sites.google.com": true, "business.site": true, "maps.google.com": true, "goo.gl": true,
}

func normalizePhoneDigits(raw string) string {
	var sb strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func (r *BusinessRepository) Upsert(ctx context.Context, lead *model.BusinessLead) (bool, error) {
	now := time.Now().UTC()
	lead.LastScrapedAt = now
	lead.UpdatedAt = now
	if lead.CreatedAt.IsZero() {
		lead.CreatedAt = now
	}

	cleanPlaceID := strings.TrimSpace(lead.PlaceID)
	cleanName := strings.TrimSpace(lead.Name)
	cleanCity := strings.TrimSpace(lead.City)
	cleanAddress := strings.TrimSpace(lead.Address)
	cleanWebsite := strings.TrimSpace(lead.Website)
	cleanDomain := normalizeDomain(cleanWebsite)
	cleanPhone := strings.TrimSpace(lead.PrimaryPhone)
	cleanPhoneDigits := normalizePhoneDigits(cleanPhone)

	if cleanPlaceID == "" {
		if cleanName != "" && cleanAddress != "" {
			cleanPlaceID = fmt.Sprintf("lead_%x", sha256.Sum256([]byte(strings.ToLower(cleanName+"|"+cleanAddress))))
		} else if cleanName != "" {
			cleanPlaceID = fmt.Sprintf("lead_%x", sha256.Sum256([]byte(strings.ToLower(cleanName+"|"+cleanCity))))
		} else {
			cleanPlaceID = fmt.Sprintf("lead_%s", uuid.New().String())
		}
		lead.PlaceID = cleanPlaceID
	}

	// Shared hosts (social pages, link-in-bio, site builders) are not a business identity.
	if sharedWebsiteHosts[cleanDomain] {
		cleanDomain = ""
	}
	// Very short digit strings are too ambiguous to identify a business.
	if len(cleanPhoneDigits) < 7 {
		cleanPhoneDigits = ""
	}

	// Step 1: Check for existing matching lead by place_id, website domain, phone digits, or name+location.
	// All comparisons are strict equality on non-empty values, and the strongest match wins.
	// (A previous version used `$3 ILIKE '%' || <existing domain> || '%'`, which is always true
	// when the existing row has no website, so new businesses overwrote unrelated rows — including
	// their `contacted` flag, which made freshly scraped leads appear already checked.)
	findQuery := `
		SELECT id, place_id, cid, name, category, subcategories, address, street, city, state,
		       postal_code, country, latitude, longitude, rating, review_count, price_level,
		       claimed, business_status, website, primary_phone, emails, phones, facebook_url,
		       instagram_url, linkedin_url, twitter_url, youtube_url, tiktok_url, google_maps_url,
		       opening_hours, photos, enrichment_status, created_at, scrape_keywords
		FROM business_leads
		WHERE ($1::text != '' AND place_id = $1::text)
		   OR ($2::text != '' AND
		       regexp_replace(regexp_replace(regexp_replace(LOWER(TRIM(COALESCE(website, ''))),
		           '^https?://', ''), '^www\.', ''), '[/?#].*$', '') = $2::text)
		   OR ($3::text != '' AND regexp_replace(COALESCE(primary_phone, ''), '[^0-9]', '', 'g') = $3::text)
		   OR ($4::text != '' AND LOWER(TRIM(name)) = LOWER($4::text) AND (
		           ($5::text != '' AND LOWER(TRIM(city)) = LOWER($5::text))
		        OR ($6::text != '' AND LOWER(TRIM(address)) = LOWER($6::text))
		   ))
		ORDER BY CASE
		    WHEN $1::text != '' AND place_id = $1::text THEN 0
		    WHEN $4::text != '' AND LOWER(TRIM(name)) = LOWER($4::text) THEN 1
		    ELSE 2
		END, created_at
		LIMIT 1
	`

	var existing model.BusinessLead
	var subcatBytes, emailsBytes, phonesBytes, hoursBytes, photosBytes, scrapeKwBytes []byte

	err := r.db.QueryRow(ctx, findQuery, cleanPlaceID, cleanDomain, cleanPhoneDigits, cleanName, cleanCity, cleanAddress).Scan(
		&existing.ID, &existing.PlaceID, &existing.CID, &existing.Name, &existing.Category, &subcatBytes,
		&existing.Address, &existing.Street, &existing.City, &existing.State, &existing.PostalCode,
		&existing.Country, &existing.Latitude, &existing.Longitude, &existing.Rating, &existing.ReviewCount,
		&existing.PriceLevel, &existing.Claimed, &existing.BusinessStatus, &existing.Website,
		&existing.PrimaryPhone, &emailsBytes, &phonesBytes, &existing.FacebookURL, &existing.InstagramURL,
		&existing.LinkedinURL, &existing.TwitterURL, &existing.YoutubeURL, &existing.TiktokURL,
		&existing.GoogleMapsURL, &hoursBytes, &photosBytes, &existing.EnrichmentStatus, &existing.CreatedAt,
		&scrapeKwBytes,
	)

	if err == nil {
		// Existing record found! Update and merge to prevent duplicates.
		lead.ID = existing.ID
		lead.CreatedAt = existing.CreatedAt

		var existingEmails, existingPhones, existingSubcats, existingScrapeKws []string
		_ = json.Unmarshal(emailsBytes, &existingEmails)
		_ = json.Unmarshal(phonesBytes, &existingPhones)
		_ = json.Unmarshal(subcatBytes, &existingSubcats)
		_ = json.Unmarshal(scrapeKwBytes, &existingScrapeKws)

		lead.Emails = mergeUniqueStrings(existingEmails, lead.Emails)
		lead.Phones = mergeUniqueStrings(existingPhones, lead.Phones)
		lead.Subcategories = mergeUniqueStrings(existingSubcats, lead.Subcategories)
		lead.ScrapeKeywords = mergeUniqueStrings(existingScrapeKws, lead.ScrapeKeywords)

		// Preserve non-empty fields if new one is blank
		if lead.PlaceID == "" {
			lead.PlaceID = existing.PlaceID
		}
		if lead.CID == "" {
			lead.CID = existing.CID
		}
		if lead.Website == "" {
			lead.Website = existing.Website
		}
		if lead.PrimaryPhone == "" {
			lead.PrimaryPhone = existing.PrimaryPhone
		}
		if lead.FacebookURL == "" {
			lead.FacebookURL = existing.FacebookURL
		}
		if lead.InstagramURL == "" {
			lead.InstagramURL = existing.InstagramURL
		}
		if lead.LinkedinURL == "" {
			lead.LinkedinURL = existing.LinkedinURL
		}
		if lead.TwitterURL == "" {
			lead.TwitterURL = existing.TwitterURL
		}
		if lead.YoutubeURL == "" {
			lead.YoutubeURL = existing.YoutubeURL
		}
		if lead.TiktokURL == "" {
			lead.TiktokURL = existing.TiktokURL
		}
		if lead.GoogleMapsURL == "" {
			lead.GoogleMapsURL = existing.GoogleMapsURL
		}
		if lead.EnrichmentStatus == model.EnrichmentNone && existing.EnrichmentStatus != model.EnrichmentNone {
			lead.EnrichmentStatus = existing.EnrichmentStatus
		}

		subcatJSON := safeJSONArray(lead.Subcategories)
		scrapeKwJSON := safeJSONArray(lead.ScrapeKeywords)
		emailsJSON := safeJSONArray(lead.Emails)
		phonesJSON := safeJSONArray(lead.Phones)
		hoursJSON := safeJSONObject(lead.OpeningHours)
		if len(hoursJSON) <= 2 && len(hoursBytes) > 2 {
			hoursJSON = hoursBytes
		}
		photosJSON := safeJSONArray(lead.Photos)
		if len(photosJSON) <= 2 && len(photosBytes) > 2 {
			photosJSON = photosBytes
		}

		updateQuery := `
			UPDATE business_leads SET
				job_id = COALESCE($2, job_id),
				place_id = $3,
				cid = $4,
				name = $5,
				category = $6,
				subcategories = $7,
				address = $8,
				street = $9,
				city = $10,
				state = $11,
				postal_code = $12,
				country = $13,
				latitude = $14,
				longitude = $15,
				rating = $16,
				review_count = $17,
				price_level = $18,
				claimed = $19,
				business_status = $20,
				website = $21,
				primary_phone = $22,
				emails = $23,
				phones = $24,
				facebook_url = $25,
				instagram_url = $26,
				linkedin_url = $27,
				twitter_url = $28,
				youtube_url = $29,
				tiktok_url = $30,
				google_maps_url = $31,
				opening_hours = $32,
				photos = $33,
				enrichment_status = $34,
				last_scraped_at = $35,
				updated_at = $36,
				scrape_keywords = $37
			WHERE id = $1
		`

		_, updateErr := r.db.Exec(ctx, updateQuery,
			lead.ID, lead.JobID, lead.PlaceID, lead.CID, lead.Name, lead.Category, subcatJSON, lead.Address, lead.Street,
			lead.City, lead.State, lead.PostalCode, lead.Country, lead.Latitude, lead.Longitude, lead.Rating, lead.ReviewCount,
			lead.PriceLevel, lead.Claimed, lead.BusinessStatus, lead.Website, lead.PrimaryPhone, emailsJSON, phonesJSON,
			lead.FacebookURL, lead.InstagramURL, lead.LinkedinURL, lead.TwitterURL, lead.YoutubeURL, lead.TiktokURL,
			lead.GoogleMapsURL, hoursJSON, photosJSON, lead.EnrichmentStatus, lead.LastScrapedAt, lead.UpdatedAt,
			scrapeKwJSON,
		)
		return false, updateErr
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}

	// Step 2: No existing match found -> Insert new lead
	if lead.ID == uuid.Nil {
		lead.ID = uuid.New()
	}

	subcatJSON := safeJSONArray(lead.Subcategories)
	scrapeKwJSON := safeJSONArray(lead.ScrapeKeywords)
	emailsJSON := safeJSONArray(lead.Emails)
	phonesJSON := safeJSONArray(lead.Phones)
	hoursJSON := safeJSONObject(lead.OpeningHours)
	photosJSON := safeJSONArray(lead.Photos)

	insertQuery := `
		INSERT INTO business_leads (
			id, job_id, place_id, cid, name, category, subcategories, address, street,
			city, state, postal_code, country, latitude, longitude, rating, review_count,
			price_level, claimed, business_status, website, primary_phone, emails, phones,
			facebook_url, instagram_url, linkedin_url, twitter_url, youtube_url, tiktok_url,
			google_maps_url, opening_hours, photos, enrichment_status, last_scraped_at,
			last_enriched_at, created_at, updated_at, scrape_keywords
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14, $15, $16, $17,
			$18, $19, $20, $21, $22, $23, $24,
			$25, $26, $27, $28, $29, $30,
			$31, $32, $33, $34, $35,
			$36, $37, $38, $39
		)
		ON CONFLICT (place_id) DO UPDATE SET
			job_id = EXCLUDED.job_id,
			cid = COALESCE(EXCLUDED.cid, business_leads.cid),
			name = EXCLUDED.name,
			category = EXCLUDED.category,
			subcategories = EXCLUDED.subcategories,
			address = EXCLUDED.address,
			street = EXCLUDED.street,
			city = EXCLUDED.city,
			state = EXCLUDED.state,
			postal_code = EXCLUDED.postal_code,
			country = EXCLUDED.country,
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			rating = EXCLUDED.rating,
			review_count = EXCLUDED.review_count,
			price_level = EXCLUDED.price_level,
			claimed = EXCLUDED.claimed,
			business_status = EXCLUDED.business_status,
			website = COALESCE(NULLIF(EXCLUDED.website, ''), business_leads.website),
			primary_phone = COALESCE(NULLIF(EXCLUDED.primary_phone, ''), business_leads.primary_phone),
			google_maps_url = EXCLUDED.google_maps_url,
			opening_hours = EXCLUDED.opening_hours,
			photos = EXCLUDED.photos,
			last_scraped_at = EXCLUDED.last_scraped_at,
			updated_at = EXCLUDED.updated_at,
			scrape_keywords = COALESCE((
				SELECT jsonb_agg(DISTINCT k)
				FROM jsonb_array_elements_text(COALESCE(business_leads.scrape_keywords, '[]'::jsonb) || EXCLUDED.scrape_keywords) AS k
			), '[]'::jsonb)
	`

	_, err = r.db.Exec(ctx, insertQuery,
		lead.ID, lead.JobID, lead.PlaceID, lead.CID, lead.Name, lead.Category, subcatJSON, lead.Address, lead.Street,
		lead.City, lead.State, lead.PostalCode, lead.Country, lead.Latitude, lead.Longitude, lead.Rating, lead.ReviewCount,
		lead.PriceLevel, lead.Claimed, lead.BusinessStatus, lead.Website, lead.PrimaryPhone, emailsJSON, phonesJSON,
		lead.FacebookURL, lead.InstagramURL, lead.LinkedinURL, lead.TwitterURL, lead.YoutubeURL, lead.TiktokURL,
		lead.GoogleMapsURL, hoursJSON, photosJSON, lead.EnrichmentStatus, lead.LastScrapedAt,
		lead.LastEnrichedAt, lead.CreatedAt, lead.UpdatedAt, scrapeKwJSON,
	)
	if err != nil {
		return false, err
	}
	return true, nil
}

func mergeUniqueStrings(a, b []string) []string {
	seen := make(map[string]bool)
	var res []string
	for _, s := range append(a, b...) {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" && !seen[trimmed] {
			seen[trimmed] = true
			res = append(res, trimmed)
		}
	}
	return res
}

func (r *BusinessRepository) UpdateEnrichment(ctx context.Context, id uuid.UUID, emails, phones []string, fb, ig, li, tw, yt, tt string, status model.EnrichmentStatus) error {
	emailsJSON := safeJSONArray(emails)
	phonesJSON := safeJSONArray(phones)
	now := time.Now().UTC()

	query := `
		UPDATE business_leads
		SET emails = $1, phones = $2, facebook_url = $3, instagram_url = $4,
		    linkedin_url = $5, twitter_url = $6, youtube_url = $7, tiktok_url = $8,
		    enrichment_status = $9, last_enriched_at = $10, updated_at = $11
		WHERE id = $12
	`
	_, err := r.db.Exec(ctx, query, emailsJSON, phonesJSON, fb, ig, li, tw, yt, tt, status, now, now, id)
	return err
}

func (r *BusinessRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.BusinessLead, error) {
	query := `
		SELECT id, job_id, place_id, cid, name, category, subcategories, address, street,
		       city, state, postal_code, country, latitude, longitude, rating, review_count,
		       price_level, claimed, business_status, website, primary_phone, emails, phones,
		       facebook_url, instagram_url, linkedin_url, twitter_url, youtube_url, tiktok_url,
		       google_maps_url, opening_hours, photos, enrichment_status, last_scraped_at,
		       last_enriched_at, contacted, contacted_at, lead_status, created_at, updated_at, scrape_keywords
		FROM business_leads
		WHERE id = $1
	`
	var lead model.BusinessLead
	var subcatBytes, emailsBytes, phonesBytes, hoursBytes, photosBytes, scrapeKwBytes []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&lead.ID, &lead.JobID, &lead.PlaceID, &lead.CID, &lead.Name, &lead.Category, &subcatBytes, &lead.Address, &lead.Street,
		&lead.City, &lead.State, &lead.PostalCode, &lead.Country, &lead.Latitude, &lead.Longitude, &lead.Rating, &lead.ReviewCount,
		&lead.PriceLevel, &lead.Claimed, &lead.BusinessStatus, &lead.Website, &lead.PrimaryPhone, &emailsBytes, &phonesBytes,
		&lead.FacebookURL, &lead.InstagramURL, &lead.LinkedinURL, &lead.TwitterURL, &lead.YoutubeURL, &lead.TiktokURL,
		&lead.GoogleMapsURL, &hoursBytes, &photosBytes, &lead.EnrichmentStatus, &lead.LastScrapedAt,
		&lead.LastEnrichedAt, &lead.Contacted, &lead.ContactedAt, &lead.LeadStatus, &lead.CreatedAt, &lead.UpdatedAt, &scrapeKwBytes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	_ = json.Unmarshal(subcatBytes, &lead.Subcategories)
	_ = json.Unmarshal(scrapeKwBytes, &lead.ScrapeKeywords)
	_ = json.Unmarshal(emailsBytes, &lead.Emails)
	_ = json.Unmarshal(phonesBytes, &lead.Phones)
	_ = json.Unmarshal(hoursBytes, &lead.OpeningHours)
	_ = json.Unmarshal(photosBytes, &lead.Photos)

	return &lead, nil
}

func (r *BusinessRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM business_leads WHERE id = $1", id)
	return err
}

// buildBusinessWhere turns a BusinessFilter into a SQL WHERE expression and its positional args.
func buildBusinessWhere(filter model.BusinessFilter) (string, []interface{}) {
	whereClauses := []string{"1=1"}
	args := []interface{}{}
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

	whereSQL, args := buildBusinessWhere(filter)
	argIdx := len(args) + 1

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM business_leads WHERE %s", whereSQL)
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

	selectQuery := fmt.Sprintf(`
		SELECT id, job_id, place_id, cid, name, category, subcategories, address, street,
		       city, state, postal_code, country, latitude, longitude, rating, review_count,
		       price_level, claimed, business_status, website, primary_phone, emails, phones,
		       facebook_url, instagram_url, linkedin_url, twitter_url, youtube_url, tiktok_url,
		       google_maps_url, opening_hours, photos, enrichment_status, last_scraped_at,
		       last_enriched_at, contacted, contacted_at, lead_status, created_at, updated_at, scrape_keywords
		FROM business_leads
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereSQL, sortBy, sortOrder, argIdx, argIdx+1)

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

		if err := rows.Scan(
			&lead.ID, &lead.JobID, &lead.PlaceID, &lead.CID, &lead.Name, &lead.Category, &subcatBytes, &lead.Address, &lead.Street,
			&lead.City, &lead.State, &lead.PostalCode, &lead.Country, &lead.Latitude, &lead.Longitude, &lead.Rating, &lead.ReviewCount,
			&lead.PriceLevel, &lead.Claimed, &lead.BusinessStatus, &lead.Website, &lead.PrimaryPhone, &emailsBytes, &phonesBytes,
			&lead.FacebookURL, &lead.InstagramURL, &lead.LinkedinURL, &lead.TwitterURL, &lead.YoutubeURL, &lead.TiktokURL,
			&lead.GoogleMapsURL, &hoursBytes, &photosBytes, &lead.EnrichmentStatus, &lead.LastScrapedAt,
			&lead.LastEnrichedAt, &lead.Contacted, &lead.ContactedAt, &lead.LeadStatus, &lead.CreatedAt, &lead.UpdatedAt, &scrapeKwBytes,
		); err != nil {
			return nil, err
		}

		_ = json.Unmarshal(subcatBytes, &lead.Subcategories)
		_ = json.Unmarshal(scrapeKwBytes, &lead.ScrapeKeywords)
		_ = json.Unmarshal(emailsBytes, &lead.Emails)
		_ = json.Unmarshal(phonesBytes, &lead.Phones)
		_ = json.Unmarshal(hoursBytes, &lead.OpeningHours)
		_ = json.Unmarshal(photosBytes, &lead.Photos)

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

func (r *BusinessRepository) GetOverviewMetrics(ctx context.Context) (*model.OverviewMetrics, error) {
	var metrics model.OverviewMetrics

	// Total leads
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM business_leads").Scan(&metrics.TotalLeads)

	// Leads with email (check non-empty emails jsonb array)
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM business_leads WHERE emails IS NOT NULL AND jsonb_typeof(emails) = 'array' AND jsonb_array_length(emails) > 0").Scan(&metrics.LeadsWithEmail)

	// Leads with phone (check primary_phone or non-empty phones jsonb array)
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM business_leads WHERE (primary_phone IS NOT NULL AND primary_phone != '') OR (phones IS NOT NULL AND jsonb_typeof(phones) = 'array' AND jsonb_array_length(phones) > 0)").Scan(&metrics.LeadsWithPhone)

	// Enriched count (websites crawled successfully)
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM business_leads WHERE enrichment_status = 'SUCCESS'").Scan(&metrics.EnrichedLeads)

	// Jobs stats
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM scrape_jobs WHERE status = 'PROCESSING'").Scan(&metrics.ActiveJobs)
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM scrape_jobs WHERE status = 'COMPLETED'").Scan(&metrics.CompletedJobs)
	_ = r.db.QueryRow(ctx, "SELECT COALESCE(SUM(total_scraped), 0) FROM scrape_jobs").Scan(&metrics.TotalScrapedEver)

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
	return stats, nil
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
	return stats, nil
}

// SetContacted ticks or unticks a lead. Ticking starts it as cold (keeping any status it
// already has); unticking clears the status so it leaves the Leads pipeline.
func (r *BusinessRepository) SetContacted(ctx context.Context, id uuid.UUID, contacted bool) error {
	var err error
	if contacted {
		_, err = r.db.Exec(ctx, `
			UPDATE business_leads
			SET contacted = true,
			    contacted_at = COALESCE(contacted_at, NOW()),
			    lead_status = COALESCE(lead_status, 'cold'),
			    updated_at = NOW()
			WHERE id = $1`, id)
	} else {
		_, err = r.db.Exec(ctx, `
			UPDATE business_leads
			SET contacted = false, contacted_at = NULL, lead_status = NULL, updated_at = NOW()
			WHERE id = $1`, id)
	}
	return err
}

// SetLeadStatus sets cold/warm/hot. A lead with a status is always contacted.
// Returns false when no lead has that id.
func (r *BusinessRepository) SetLeadStatus(ctx context.Context, id uuid.UUID, status model.LeadStatus) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE business_leads
		SET lead_status = $1,
		    contacted = true,
		    contacted_at = COALESCE(contacted_at, NOW()),
		    updated_at = NOW()
		WHERE id = $2`, string(status), id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
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
	return keywords, nil
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

// ListPendingEnrichment returns leads that should have been enriched but never were (their crawl was
// interrupted): they have a website, enrichment status NONE or PENDING, were scraped more than
// olderThan ago, and their job asked for enrichment.
func (r *BusinessRepository) ListPendingEnrichment(ctx context.Context, olderThan time.Duration, limit int) ([]*model.BusinessLead, error) {
	rows, err := r.db.Query(ctx, `
		SELECT bl.id, bl.name, bl.website, bl.job_id
		FROM business_leads bl
		JOIN scrape_jobs j ON j.id = bl.job_id
		WHERE bl.enrichment_status IN ('NONE', 'PENDING')
		  AND COALESCE(bl.website, '') <> ''
		  AND j.enrich_contacts = true
		  AND bl.last_scraped_at < $1
		ORDER BY bl.last_scraped_at
		LIMIT $2`, time.Now().UTC().Add(-olderThan), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leads []*model.BusinessLead
	for rows.Next() {
		l := &model.BusinessLead{EnrichmentStatus: model.EnrichmentNone}
		if err := rows.Scan(&l.ID, &l.Name, &l.Website, &l.JobID); err != nil {
			return nil, err
		}
		leads = append(leads, l)
	}
	return leads, rows.Err()
}
