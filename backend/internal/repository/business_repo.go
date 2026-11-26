package repository

import (
	"business-scraping-service/backend/internal/model"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

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

func safeJSONObject(obj map[string]any) []byte {
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

		// These lists are merged and written back: if one can't be read, stop rather than
		// overwrite the stored values with only the new ones.
		var existingEmails, existingPhones, existingSubcats, existingScrapeKws []string
		for _, c := range []struct {
			name string
			raw  []byte
			dst  *[]string
		}{
			{"emails", emailsBytes, &existingEmails},
			{"phones", phonesBytes, &existingPhones},
			{"subcategories", subcatBytes, &existingSubcats},
			{"scrape_keywords", scrapeKwBytes, &existingScrapeKws},
		} {
			if err := decodeJSON(c.raw, c.dst); err != nil {
				return false, fmt.Errorf("lead %s: reading stored %s: %w", existing.ID, c.name, err)
			}
		}

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
		       last_enriched_at, contacted, contacted_at, lead_status, created_at, updated_at, scrape_keywords,
		       group_type, group_name, suggested_group_type, suggested_group_name,
		       contacted_by, COALESCE((SELECT username FROM admin_users WHERE id = business_leads.contacted_by), '')
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
		&lead.GroupType, &lead.GroupName, &lead.SuggestedGroupType, &lead.SuggestedGroupName,
		&lead.ContactedBy, &lead.ContactedByName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	decodeForDisplay(subcatBytes, &lead.Subcategories, "business_leads", lead.ID, "subcategories")
	decodeForDisplay(scrapeKwBytes, &lead.ScrapeKeywords, "business_leads", lead.ID, "scrape_keywords")
	decodeForDisplay(emailsBytes, &lead.Emails, "business_leads", lead.ID, "emails")
	decodeForDisplay(phonesBytes, &lead.Phones, "business_leads", lead.ID, "phones")
	decodeForDisplay(hoursBytes, &lead.OpeningHours, "business_leads", lead.ID, "opening_hours")
	decodeForDisplay(photosBytes, &lead.Photos, "business_leads", lead.ID, "photos")

	return &lead, nil
}

func (r *BusinessRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM business_leads WHERE id = $1", id)
	return err
}

// SetContacted ticks or unticks a lead. Ticking starts it as cold (keeping any status it
// already has); unticking clears the status so it leaves the Leads pipeline.
// SetContacted ticks or unticks a lead. by is the admin doing it: the first admin to mark a
// lead contacted is kept as contacted_by, like contacted_at; unticking clears both.
func (r *BusinessRepository) SetContacted(ctx context.Context, id uuid.UUID, contacted bool, by *uuid.UUID) error {
	var err error
	if contacted {
		_, err = r.db.Exec(ctx, `
			UPDATE business_leads
			SET contacted = true,
			    contacted_at = COALESCE(contacted_at, NOW()),
			    contacted_by = CASE WHEN contacted THEN contacted_by ELSE $2 END,
			    lead_status = COALESCE(lead_status, 'cold'),
			    updated_at = NOW()
			WHERE id = $1`, id, by)
	} else {
		_, err = r.db.Exec(ctx, `
			UPDATE business_leads
			SET contacted = false, contacted_at = NULL, contacted_by = NULL, lead_status = NULL, updated_at = NOW()
			WHERE id = $1`, id)
	}
	return err
}

// SetLeadStatus sets cold/warm/hot. A lead with a status is always contacted.
// Returns false when no lead has that id.
func (r *BusinessRepository) SetLeadStatus(ctx context.Context, id uuid.UUID, status model.LeadStatus, by *uuid.UUID) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE business_leads
		SET lead_status = $1,
		    contacted_by = CASE WHEN contacted THEN contacted_by ELSE $3 END,
		    contacted = true,
		    contacted_at = COALESCE(contacted_at, NOW()),
		    updated_at = NOW()
		WHERE id = $2`, string(status), id, by)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
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
