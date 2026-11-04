package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"business-scraping-service/backend/internal/model"
)

type KeywordRepository struct {
	db *pgxpool.Pool
}

func NewKeywordRepository(db *pgxpool.Pool) *KeywordRepository {
	return &KeywordRepository{db: db}
}

func (r *KeywordRepository) List(ctx context.Context, filter model.ListKeywordFilter) ([]*model.ScraperKeyword, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 || filter.PageSize > 100 {
		filter.PageSize = 50
	}

	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if strings.TrimSpace(filter.Search) != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("keyword ILIKE $%d", argIdx))
		args = append(args, "%"+strings.TrimSpace(filter.Search)+"%")
		argIdx++
	}

	if strings.TrimSpace(filter.Status) != "" && filter.Status != "all" {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, strings.TrimSpace(filter.Status))
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM scraper_keywords WHERE %s", whereSQL)
	var totalCount int64
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count scraper keywords: %w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize
	query := fmt.Sprintf(`
		SELECT id, keyword, COALESCE(next_page_token, ''), current_page,
		       COALESCE(
		           (
		               SELECT COUNT(DISTINCT bl.id)
		               FROM business_leads bl
		               WHERE bl.subcategories @> jsonb_build_array(scraper_keywords.keyword)
		                  OR bl.job_id IN (SELECT sj.id FROM scrape_jobs sj WHERE sj.query = scraper_keywords.keyword)
		           ),
		           0
		       ) AS total_leads_scraped,
		       last_scraped_at, status, is_active, created_at, updated_at
		FROM scraper_keywords
		WHERE %s
		ORDER BY updated_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, filter.PageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query scraper keywords: %w", err)
	}
	defer rows.Close()

	var items []*model.ScraperKeyword
	for rows.Next() {
		kw := &model.ScraperKeyword{}
		if err := rows.Scan(
			&kw.ID,
			&kw.Keyword,
			&kw.NextPageToken,
			&kw.CurrentPage,
			&kw.TotalLeadsScraped,
			&kw.LastScrapedAt,
			&kw.Status,
			&kw.IsActive,
			&kw.CreatedAt,
			&kw.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan scraper keyword: %w", err)
		}
		items = append(items, kw)
	}

	return items, totalCount, nil
}

func (r *KeywordRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ScraperKeyword, error) {
	query := `
		SELECT id, keyword, COALESCE(next_page_token, ''), current_page,
		       COALESCE(
		           (
		               SELECT COUNT(DISTINCT bl.id)
		               FROM business_leads bl
		               WHERE bl.subcategories @> jsonb_build_array(scraper_keywords.keyword)
		                  OR bl.job_id IN (SELECT sj.id FROM scrape_jobs sj WHERE sj.query = scraper_keywords.keyword)
		           ),
		           0
		       ) AS total_leads_scraped,
		       last_scraped_at, status, is_active, created_at, updated_at
		FROM scraper_keywords
		WHERE id = $1
	`
	kw := &model.ScraperKeyword{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&kw.ID,
		&kw.Keyword,
		&kw.NextPageToken,
		&kw.CurrentPage,
		&kw.TotalLeadsScraped,
		&kw.LastScrapedAt,
		&kw.Status,
		&kw.IsActive,
		&kw.CreatedAt,
		&kw.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return kw, nil
}

func (r *KeywordRepository) Create(ctx context.Context, keyword string) (*model.ScraperKeyword, error) {
	cleaned := strings.TrimSpace(keyword)
	if cleaned == "" {
		return nil, fmt.Errorf("keyword cannot be empty")
	}

	query := `
		INSERT INTO scraper_keywords (keyword, status, is_active, created_at, updated_at)
		VALUES ($1, 'ready', false, NOW(), NOW())
		ON CONFLICT (keyword) DO UPDATE SET updated_at = NOW()
		RETURNING id, keyword, COALESCE(next_page_token, ''), current_page, total_leads_scraped,
		          last_scraped_at, status, is_active, created_at, updated_at
	`
	kw := &model.ScraperKeyword{}
	err := r.db.QueryRow(ctx, query, cleaned).Scan(
		&kw.ID,
		&kw.Keyword,
		&kw.NextPageToken,
		&kw.CurrentPage,
		&kw.TotalLeadsScraped,
		&kw.LastScrapedAt,
		&kw.Status,
		&kw.IsActive,
		&kw.CreatedAt,
		&kw.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create scraper keyword: %w", err)
	}
	return kw, nil
}

func (r *KeywordRepository) CreateBulk(ctx context.Context, keywords []string) (int, error) {
	count := 0
	for _, raw := range keywords {
		cleaned := strings.TrimSpace(raw)
		if cleaned == "" {
			continue
		}
		_, err := r.Create(ctx, cleaned)
		if err == nil {
			count++
		}
	}
	return count, nil
}

func (r *KeywordRepository) UpdateCursor(ctx context.Context, id uuid.UUID, nextToken string, currentPage int, addLeads int, status model.KeywordStatus) error {
	now := time.Now().UTC()
	query := `
		UPDATE scraper_keywords
		SET next_page_token = $1,
		    current_page = $2,
		    total_leads_scraped = COALESCE(
		        (
		            SELECT COUNT(DISTINCT bl.id)
		            FROM business_leads bl
		            WHERE bl.subcategories @> jsonb_build_array(scraper_keywords.keyword)
		               OR bl.job_id IN (SELECT sj.id FROM scrape_jobs sj WHERE sj.query = scraper_keywords.keyword)
		        ),
		        total_leads_scraped + $3
		    ),
		    status = $4,
		    last_scraped_at = $5,
		    updated_at = $5
		WHERE id = $6
	`
	_, err := r.db.Exec(ctx, query, nextToken, currentPage, addLeads, status, now, id)
	return err
}

func (r *KeywordRepository) SetStatus(ctx context.Context, id uuid.UUID, status model.KeywordStatus) error {
	query := `UPDATE scraper_keywords SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, status, id)
	return err
}

func (r *KeywordRepository) ResetCursor(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE scraper_keywords
		SET next_page_token = '',
		    current_page = 1,
		    status = 'ready',
		    updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *KeywordRepository) GetByKeyword(ctx context.Context, keyword string) (*model.ScraperKeyword, error) {
	cleaned := strings.TrimSpace(keyword)
	if cleaned == "" {
		return nil, fmt.Errorf("keyword cannot be empty")
	}

	query := `
		SELECT id, keyword, COALESCE(next_page_token, ''), current_page,
		       COALESCE(
		           (
		               SELECT COUNT(DISTINCT bl.id)
		               FROM business_leads bl
		               WHERE bl.subcategories @> jsonb_build_array(scraper_keywords.keyword)
		                  OR bl.job_id IN (SELECT sj.id FROM scrape_jobs sj WHERE sj.query = scraper_keywords.keyword)
		           ),
		           0
		       ) AS total_leads_scraped,
		       last_scraped_at, status, is_active, created_at, updated_at
		FROM scraper_keywords
		WHERE LOWER(TRIM(keyword)) = LOWER(TRIM($1))
		LIMIT 1
	`
	kw := &model.ScraperKeyword{}
	err := r.db.QueryRow(ctx, query, cleaned).Scan(
		&kw.ID,
		&kw.Keyword,
		&kw.NextPageToken,
		&kw.CurrentPage,
		&kw.TotalLeadsScraped,
		&kw.LastScrapedAt,
		&kw.Status,
		&kw.IsActive,
		&kw.CreatedAt,
		&kw.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return kw, nil
}

func (r *KeywordRepository) Update(ctx context.Context, id uuid.UUID, keyword string, isActive bool) (*model.ScraperKeyword, error) {
	now := time.Now().UTC()
	cleaned := strings.TrimSpace(keyword)

	// If the keyword text has changed, the Google Places pagination token is invalid
	// and must be reset to avoid 400 INVALID_ARGUMENT.
	query := `
		UPDATE scraper_keywords
		SET keyword = $1,
		    is_active = $2,
		    next_page_token = CASE WHEN LOWER(TRIM(keyword)) != LOWER(TRIM($1)) THEN '' ELSE next_page_token END,
		    current_page = CASE WHEN LOWER(TRIM(keyword)) != LOWER(TRIM($1)) THEN 1 ELSE current_page END,
		    status = CASE WHEN LOWER(TRIM(keyword)) != LOWER(TRIM($1)) THEN 'ready' ELSE status END,
		    updated_at = $3
		WHERE id = $4
		RETURNING id, keyword, COALESCE(next_page_token, ''), current_page, total_leads_scraped,
		          last_scraped_at, status, is_active, created_at, updated_at
	`
	kw := &model.ScraperKeyword{}
	err := r.db.QueryRow(ctx, query, cleaned, isActive, now, id).Scan(
		&kw.ID,
		&kw.Keyword,
		&kw.NextPageToken,
		&kw.CurrentPage,
		&kw.TotalLeadsScraped,
		&kw.LastScrapedAt,
		&kw.Status,
		&kw.IsActive,
		&kw.CreatedAt,
		&kw.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return kw, nil
}

func (r *KeywordRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM scraper_keywords WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *KeywordRepository) GetActiveKeywords(ctx context.Context) ([]*model.ScraperKeyword, error) {
	query := `
		SELECT id, keyword, COALESCE(next_page_token, ''), current_page, total_leads_scraped,
		       last_scraped_at, status, is_active, created_at, updated_at
		FROM scraper_keywords
		WHERE is_active = true
		ORDER BY COALESCE(last_scraped_at, '1970-01-01'::timestamptz) ASC, created_at ASC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*model.ScraperKeyword
	for rows.Next() {
		kw := &model.ScraperKeyword{}
		if err := rows.Scan(
			&kw.ID,
			&kw.Keyword,
			&kw.NextPageToken,
			&kw.CurrentPage,
			&kw.TotalLeadsScraped,
			&kw.LastScrapedAt,
			&kw.Status,
			&kw.IsActive,
			&kw.CreatedAt,
			&kw.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, kw)
	}
	return items, nil
}
