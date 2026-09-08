package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"business-scraping-service/backend/internal/model"
)

type BusinessRepository struct {
	db *pgxpool.Pool
}

func NewBusinessRepository(db *pgxpool.Pool) *BusinessRepository {
	return &BusinessRepository{db: db}
}

func (r *BusinessRepository) Upsert(ctx context.Context, lead *model.BusinessLead) error {
	now := time.Now().UTC()
	lead.LastScrapedAt = now
	lead.UpdatedAt = now
	if lead.CreatedAt.IsZero() {
		lead.CreatedAt = now
	}

	cleanPlaceID := strings.TrimSpace(lead.PlaceID)
	cleanName := strings.TrimSpace(lead.Name)
	cleanCity := strings.TrimSpace(lead.City)
	cleanWebsite := strings.TrimSpace(lead.Website)
	cleanPhone := strings.TrimSpace(lead.PrimaryPhone)

	// Step 1: Check for existing matching lead by place_id, website domain, phone, or name+city
	findQuery := `
		SELECT id, place_id, cid, name, category, subcategories, address, street, city, state, 
		       postal_code, country, latitude, longitude, rating, review_count, price_level, 
		       claimed, business_status, website, primary_phone, emails, phones, facebook_url, 
		       instagram_url, linkedin_url, twitter_url, youtube_url, tiktok_url, google_maps_url, 
		       opening_hours, photos, enrichment_status, created_at
		FROM business_leads
		WHERE ($1 != '' AND place_id = $1)
		   OR ($2 != '' AND (website = $2 OR website ILIKE '%' || $2 || '%'))
		   OR ($3 != '' AND primary_phone = $3)
		   OR ($4 != '' AND $5 != '' AND LOWER(TRIM(name)) = LOWER(TRIM($4)) AND LOWER(TRIM(city)) = LOWER(TRIM($5)))
		LIMIT 1
	`

	var existing model.BusinessLead
	var subcatBytes, emailsBytes, phonesBytes, hoursBytes, photosBytes []byte

	err := r.db.QueryRow(ctx, findQuery, cleanPlaceID, cleanWebsite, cleanPhone, cleanName, cleanCity).Scan(
		&existing.ID, &existing.PlaceID, &existing.CID, &existing.Name, &existing.Category, &subcatBytes,
		&existing.Address, &existing.Street, &existing.City, &existing.State, &existing.PostalCode,
		&existing.Country, &existing.Latitude, &existing.Longitude, &existing.Rating, &existing.ReviewCount,
		&existing.PriceLevel, &existing.Claimed, &existing.BusinessStatus, &existing.Website,
		&existing.PrimaryPhone, &emailsBytes, &phonesBytes, &existing.FacebookURL, &existing.InstagramURL,
		&existing.LinkedinURL, &existing.TwitterURL, &existing.YoutubeURL, &existing.TiktokURL,
		&existing.GoogleMapsURL, &hoursBytes, &photosBytes, &existing.EnrichmentStatus, &existing.CreatedAt,
	)

	if err == nil {
		// Existing record found! Update and merge to prevent duplicates.
		lead.ID = existing.ID
		lead.CreatedAt = existing.CreatedAt

		var existingEmails, existingPhones, existingSubcats []string
		_ = json.Unmarshal(emailsBytes, &existingEmails)
		_ = json.Unmarshal(phonesBytes, &existingPhones)
		_ = json.Unmarshal(subcatBytes, &existingSubcats)

		lead.Emails = mergeUniqueStrings(existingEmails, lead.Emails)
		lead.Phones = mergeUniqueStrings(existingPhones, lead.Phones)
		lead.Subcategories = mergeUniqueStrings(existingSubcats, lead.Subcategories)

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

		subcatJSON, _ := json.Marshal(lead.Subcategories)
		emailsJSON, _ := json.Marshal(lead.Emails)
		phonesJSON, _ := json.Marshal(lead.Phones)
		hoursJSON, _ := json.Marshal(lead.OpeningHours)
		if len(hoursJSON) <= 2 && len(hoursBytes) > 2 {
			hoursJSON = hoursBytes
		}
		photosJSON, _ := json.Marshal(lead.Photos)
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
				updated_at = $36
			WHERE id = $1
		`

		_, updateErr := r.db.Exec(ctx, updateQuery,
			lead.ID, lead.JobID, lead.PlaceID, lead.CID, lead.Name, lead.Category, subcatJSON, lead.Address, lead.Street,
			lead.City, lead.State, lead.PostalCode, lead.Country, lead.Latitude, lead.Longitude, lead.Rating, lead.ReviewCount,
			lead.PriceLevel, lead.Claimed, lead.BusinessStatus, lead.Website, lead.PrimaryPhone, emailsJSON, phonesJSON,
			lead.FacebookURL, lead.InstagramURL, lead.LinkedinURL, lead.TwitterURL, lead.YoutubeURL, lead.TiktokURL,
			lead.GoogleMapsURL, hoursJSON, photosJSON, lead.EnrichmentStatus, lead.LastScrapedAt, lead.UpdatedAt,
		)
		return updateErr
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	// Step 2: No existing match found -> Insert new lead
	if lead.ID == uuid.Nil {
		lead.ID = uuid.New()
	}

	subcatJSON, _ := json.Marshal(lead.Subcategories)
	if len(subcatJSON) == 0 {
		subcatJSON = []byte("[]")
	}
	emailsJSON, _ := json.Marshal(lead.Emails)
	if len(emailsJSON) == 0 {
		emailsJSON = []byte("[]")
	}
	phonesJSON, _ := json.Marshal(lead.Phones)
	if len(phonesJSON) == 0 {
		phonesJSON = []byte("[]")
	}
	hoursJSON, _ := json.Marshal(lead.OpeningHours)
	if len(hoursJSON) == 0 {
		hoursJSON = []byte("{}")
	}
	photosJSON, _ := json.Marshal(lead.Photos)
	if len(photosJSON) == 0 {
		photosJSON = []byte("[]")
	}

	insertQuery := `
		INSERT INTO business_leads (
			id, job_id, place_id, cid, name, category, subcategories, address, street,
			city, state, postal_code, country, latitude, longitude, rating, review_count,
			price_level, claimed, business_status, website, primary_phone, emails, phones,
			facebook_url, instagram_url, linkedin_url, twitter_url, youtube_url, tiktok_url,
			google_maps_url, opening_hours, photos, enrichment_status, last_scraped_at,
			last_enriched_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14, $15, $16, $17,
			$18, $19, $20, $21, $22, $23, $24,
			$25, $26, $27, $28, $29, $30,
			$31, $32, $33, $34, $35,
			$36, $37, $38
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
			updated_at = EXCLUDED.updated_at
	`

	_, err = r.db.Exec(ctx, insertQuery,
		lead.ID, lead.JobID, lead.PlaceID, lead.CID, lead.Name, lead.Category, subcatJSON, lead.Address, lead.Street,
		lead.City, lead.State, lead.PostalCode, lead.Country, lead.Latitude, lead.Longitude, lead.Rating, lead.ReviewCount,
		lead.PriceLevel, lead.Claimed, lead.BusinessStatus, lead.Website, lead.PrimaryPhone, emailsJSON, phonesJSON,
		lead.FacebookURL, lead.InstagramURL, lead.LinkedinURL, lead.TwitterURL, lead.YoutubeURL, lead.TiktokURL,
		lead.GoogleMapsURL, hoursJSON, photosJSON, lead.EnrichmentStatus, lead.LastScrapedAt,
		lead.LastEnrichedAt, lead.CreatedAt, lead.UpdatedAt,
	)
	return err
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
	emailsJSON, _ := json.Marshal(emails)
	if len(emailsJSON) == 0 {
		emailsJSON = []byte("[]")
	}
	phonesJSON, _ := json.Marshal(phones)
	if len(phonesJSON) == 0 {
		phonesJSON = []byte("[]")
	}
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
		       last_enriched_at, created_at, updated_at
		FROM business_leads
		WHERE id = $1
	`
	var lead model.BusinessLead
	var subcatBytes, emailsBytes, phonesBytes, hoursBytes, photosBytes []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&lead.ID, &lead.JobID, &lead.PlaceID, &lead.CID, &lead.Name, &lead.Category, &subcatBytes, &lead.Address, &lead.Street,
		&lead.City, &lead.State, &lead.PostalCode, &lead.Country, &lead.Latitude, &lead.Longitude, &lead.Rating, &lead.ReviewCount,
		&lead.PriceLevel, &lead.Claimed, &lead.BusinessStatus, &lead.Website, &lead.PrimaryPhone, &emailsBytes, &phonesBytes,
		&lead.FacebookURL, &lead.InstagramURL, &lead.LinkedinURL, &lead.TwitterURL, &lead.YoutubeURL, &lead.TiktokURL,
		&lead.GoogleMapsURL, &hoursBytes, &photosBytes, &lead.EnrichmentStatus, &lead.LastScrapedAt,
		&lead.LastEnrichedAt, &lead.CreatedAt, &lead.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	_ = json.Unmarshal(subcatBytes, &lead.Subcategories)
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

func (r *BusinessRepository) List(ctx context.Context, filter model.BusinessFilter) (*model.BusinessListResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}
	offset := (filter.Page - 1) * filter.PageSize

	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if filter.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(name ILIKE $%d OR address ILIKE $%d OR website ILIKE $%d OR primary_phone ILIKE $%d)", argIdx, argIdx, argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}
	if filter.Category != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("category ILIKE $%d", argIdx))
		args = append(args, "%"+filter.Category+"%")
		argIdx++
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
			whereClauses = append(whereClauses, "jsonb_array_length(emails) > 0")
		} else {
			whereClauses = append(whereClauses, "(emails IS NULL OR jsonb_array_length(emails) = 0)")
		}
	}
	if filter.HasPhone != nil {
		if *filter.HasPhone {
			whereClauses = append(whereClauses, "(primary_phone != '' OR jsonb_array_length(phones) > 0)")
		} else {
			whereClauses = append(whereClauses, "primary_phone = '' AND (phones IS NULL OR jsonb_array_length(phones) = 0)")
		}
	}
	if filter.JobID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("job_id = $%d", argIdx))
		args = append(args, *filter.JobID)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

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
		       last_enriched_at, created_at, updated_at
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
		var subcatBytes, emailsBytes, phonesBytes, hoursBytes, photosBytes []byte

		if err := rows.Scan(
			&lead.ID, &lead.JobID, &lead.PlaceID, &lead.CID, &lead.Name, &lead.Category, &subcatBytes, &lead.Address, &lead.Street,
			&lead.City, &lead.State, &lead.PostalCode, &lead.Country, &lead.Latitude, &lead.Longitude, &lead.Rating, &lead.ReviewCount,
			&lead.PriceLevel, &lead.Claimed, &lead.BusinessStatus, &lead.Website, &lead.PrimaryPhone, &emailsBytes, &phonesBytes,
			&lead.FacebookURL, &lead.InstagramURL, &lead.LinkedinURL, &lead.TwitterURL, &lead.YoutubeURL, &lead.TiktokURL,
			&lead.GoogleMapsURL, &hoursBytes, &photosBytes, &lead.EnrichmentStatus, &lead.LastScrapedAt,
			&lead.LastEnrichedAt, &lead.CreatedAt, &lead.UpdatedAt,
		); err != nil {
			return nil, err
		}

		_ = json.Unmarshal(subcatBytes, &lead.Subcategories)
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

func (r *BusinessRepository) GetAllForExport(ctx context.Context, filter model.BusinessFilter) ([]*model.BusinessLead, error) {
	filter.Page = 1
	filter.PageSize = 10000 // safe max export limit
	resp, err := r.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (r *BusinessRepository) GetOverviewMetrics(ctx context.Context) (*model.OverviewMetrics, error) {
	var metrics model.OverviewMetrics

	// Total leads
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM business_leads").Scan(&metrics.TotalLeads)

	// Leads with email
	var leadsWithEmail int64
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM business_leads WHERE jsonb_array_length(emails) > 0").Scan(&leadsWithEmail)

	// Leads with phone
	var leadsWithPhone int64
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM business_leads WHERE primary_phone != '' OR jsonb_array_length(phones) > 0").Scan(&leadsWithPhone)

	// Enriched count
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM business_leads WHERE enrichment_status = 'SUCCESS'").Scan(&metrics.EnrichedLeads)

	// Jobs stats
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM scrape_jobs WHERE status = 'PROCESSING'").Scan(&metrics.ActiveJobs)
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM scrape_jobs WHERE status = 'COMPLETED'").Scan(&metrics.CompletedJobs)
	_ = r.db.QueryRow(ctx, "SELECT COALESCE(SUM(total_scraped), 0) FROM scrape_jobs").Scan(&metrics.TotalScrapedEver)

	if metrics.TotalLeads > 0 {
		metrics.EmailHitRate = (float64(leadsWithEmail) / float64(metrics.TotalLeads)) * 100
		metrics.PhoneHitRate = (float64(leadsWithPhone) / float64(metrics.TotalLeads)) * 100
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
