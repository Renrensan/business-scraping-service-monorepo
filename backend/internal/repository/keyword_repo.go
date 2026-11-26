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

// keywordSelect returns keywords with their lead count, area name and cell progress.
const keywordSelect = `
	SELECT k.id, k.keyword, COALESCE(k.next_page_token, ''), k.current_page,
	       COALESCE((
	           SELECT COUNT(DISTINCT bl.id)
	           FROM business_leads bl
	           WHERE bl.scrape_keywords @> jsonb_build_array(k.keyword)
	              OR bl.job_id IN (SELECT sj.id FROM scrape_jobs sj WHERE sj.query = k.keyword)
	       ), 0) AS total_leads_scraped,
	       k.last_scraped_at, k.status, k.is_active, k.created_at, k.updated_at,
	       k.mode, k.term, k.included_type, k.area_id, COALESCE(a.name, ''),
	       (SELECT COUNT(*) FROM keyword_cells c WHERE c.keyword_id = k.id AND c.status <> 'split'),
	       (SELECT COUNT(*) FROM keyword_cells c WHERE c.keyword_id = k.id AND c.status = 'done'),
	       k.new_places_last_run, k.empty_runs, k.next_eligible_at
	FROM scraper_keywords k
	LEFT JOIN areas a ON a.id = k.area_id`

func scanKeyword(row pgx.Row) (*model.ScraperKeyword, error) {
	kw := &model.ScraperKeyword{}
	err := row.Scan(
		&kw.ID, &kw.Keyword, &kw.NextPageToken, &kw.CurrentPage, &kw.TotalLeadsScraped,
		&kw.LastScrapedAt, &kw.Status, &kw.IsActive, &kw.CreatedAt, &kw.UpdatedAt,
		&kw.Mode, &kw.Term, &kw.IncludedType, &kw.AreaID, &kw.AreaName,
		&kw.CellsTotal, &kw.CellsDone,
		&kw.NewPlacesLastRun, &kw.EmptyRuns, &kw.NextEligibleAt,
	)
	return kw, err
}

func (r *KeywordRepository) queryKeywords(ctx context.Context, sql string, args ...interface{}) ([]*model.ScraperKeyword, error) {
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*model.ScraperKeyword
	for rows.Next() {
		kw, err := scanKeyword(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scraper keyword: %w", err)
		}
		items = append(items, kw)
	}
	return items, rows.Err()
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
		whereClauses = append(whereClauses, fmt.Sprintf("k.keyword ILIKE $%d", argIdx))
		args = append(args, "%"+strings.TrimSpace(filter.Search)+"%")
		argIdx++
	}
	if strings.TrimSpace(filter.Status) != "" && filter.Status != "all" {
		whereClauses = append(whereClauses, fmt.Sprintf("k.status = $%d", argIdx))
		args = append(args, strings.TrimSpace(filter.Status))
		argIdx++
	}
	whereSQL := strings.Join(whereClauses, " AND ")

	var totalCount int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM scraper_keywords k WHERE "+whereSQL, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count scraper keywords: %w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize
	query := fmt.Sprintf(`%s WHERE %s ORDER BY k.updated_at DESC LIMIT $%d OFFSET $%d`, keywordSelect, whereSQL, argIdx, argIdx+1)
	args = append(args, filter.PageSize, offset)

	items, err := r.queryKeywords(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query scraper keywords: %w", err)
	}
	return items, totalCount, nil
}

func (r *KeywordRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ScraperKeyword, error) {
	kw, err := scanKeyword(r.db.QueryRow(ctx, keywordSelect+` WHERE k.id = $1`, id))
	if err != nil {
		return nil, err
	}
	return kw, nil
}

func (r *KeywordRepository) GetByKeyword(ctx context.Context, keyword string) (*model.ScraperKeyword, error) {
	cleaned := strings.TrimSpace(keyword)
	if cleaned == "" {
		return nil, fmt.Errorf("keyword cannot be empty")
	}
	kw, err := scanKeyword(r.db.QueryRow(ctx, keywordSelect+` WHERE LOWER(TRIM(k.keyword)) = LOWER(TRIM($1)) LIMIT 1`, cleaned))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return kw, nil
}

// Create adds a plain keyword (or returns the existing one with the same text).
func (r *KeywordRepository) Create(ctx context.Context, keyword string) (*model.ScraperKeyword, error) {
	cleaned := strings.TrimSpace(keyword)
	if cleaned == "" {
		return nil, fmt.Errorf("keyword cannot be empty")
	}
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO scraper_keywords (keyword, status, is_active, created_at, updated_at)
		VALUES ($1, 'ready', false, NOW(), NOW())
		ON CONFLICT (keyword) DO UPDATE SET updated_at = NOW()
		RETURNING id`, cleaned).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create scraper keyword: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *KeywordRepository) CreateBulk(ctx context.Context, keywords []string) (int, error) {
	count := 0
	for _, raw := range keywords {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		if _, err := r.Create(ctx, raw); err == nil {
			count++
		}
	}
	return count, nil
}

// AreaKeywordLabel is how an area keyword is shown and tagged on leads, e.g. "villa · Canggu".
func AreaKeywordLabel(term, areaName string) string {
	return strings.TrimSpace(term) + " · " + strings.TrimSpace(areaName)
}

// CreateAreaKeyword adds a term+type+area keyword. created is false if it already existed.
func (r *KeywordRepository) CreateAreaKeyword(ctx context.Context, term, includedType string, area *model.Area) (*model.ScraperKeyword, bool, error) {
	term = strings.TrimSpace(term)
	if term == "" {
		return nil, false, fmt.Errorf("term cannot be empty")
	}
	label := AreaKeywordLabel(term, area.Name)
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO scraper_keywords (keyword, status, is_active, mode, term, included_type, area_id, created_at, updated_at)
		VALUES ($1, 'ready', false, 'area', $2, $3, $4, NOW(), NOW())
		ON CONFLICT (keyword) DO NOTHING
		RETURNING id`, label, term, strings.TrimSpace(includedType), area.ID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		kw, gerr := r.GetByKeyword(ctx, label)
		return kw, false, gerr
	}
	if err != nil {
		return nil, false, fmt.Errorf("failed to create area keyword: %w", err)
	}
	kw, err := r.GetByID(ctx, id)
	return kw, true, err
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
		            WHERE bl.scrape_keywords @> jsonb_build_array(scraper_keywords.keyword)
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

// RecordRunOutcome stores how many new places a run found. Runs that find nothing new push the
// keyword's next scheduled run back (1, 2, 4 ... up to 30 days); a run with new places resets that.
// Manual "Scrape" clicks ignore the backoff.
func (r *KeywordRepository) RecordRunOutcome(ctx context.Context, id uuid.UUID, newPlaces int, now time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE scraper_keywords
		SET new_places_last_run = $2,
		    empty_runs = CASE WHEN $2 > 0 THEN 0 ELSE empty_runs + 1 END,
		    next_eligible_at = CASE WHEN $2 > 0 THEN NULL
		        ELSE $3::timestamptz + LEAST(POWER(2, empty_runs), 30) * INTERVAL '1 day' END,
		    updated_at = NOW()
		WHERE id = $1`, id, newPlaces, now)
	return err
}

func (r *KeywordRepository) SetStatus(ctx context.Context, id uuid.UUID, status model.KeywordStatus) error {
	_, err := r.db.Exec(ctx, `UPDATE scraper_keywords SET status = $1, updated_at = NOW() WHERE id = $2`, status, id)
	return err
}

// ResetCursor restarts a keyword from the beginning (page 1, or a fresh set of area cells).
func (r *KeywordRepository) ResetCursor(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM keyword_cells WHERE keyword_id = $1`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE scraper_keywords
		SET next_page_token = '', current_page = 1, status = 'ready', updated_at = NOW()
		WHERE id = $1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *KeywordRepository) Update(ctx context.Context, id uuid.UUID, keyword string, isActive bool) (*model.ScraperKeyword, error) {
	cleaned := strings.TrimSpace(keyword)
	// For plain keywords the text is the query: changing it invalidates the page token.
	// For area keywords the text is only a label; the term and area define the search.
	_, err := r.db.Exec(ctx, `
		UPDATE scraper_keywords
		SET keyword = $1::text,
		    is_active = $2,
		    next_page_token = CASE WHEN mode = 'plain' AND LOWER(TRIM(keyword)) != LOWER(TRIM($1::text)) THEN '' ELSE next_page_token END,
		    current_page = CASE WHEN mode = 'plain' AND LOWER(TRIM(keyword)) != LOWER(TRIM($1::text)) THEN 1 ELSE current_page END,
		    status = CASE WHEN mode = 'plain' AND LOWER(TRIM(keyword)) != LOWER(TRIM($1::text)) THEN 'ready' ELSE status END,
		    updated_at = $3
		WHERE id = $4`, cleaned, isActive, time.Now().UTC(), id)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *KeywordRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM scraper_keywords WHERE id = $1`, id)
	return err
}

// GetActiveKeywords returns active keywords for the scheduler: due ones only (not backed off),
// those that found the most new places last time first.
func (r *KeywordRepository) GetActiveKeywords(ctx context.Context) ([]*model.ScraperKeyword, error) {
	return r.queryKeywords(ctx, keywordSelect+`
		WHERE k.is_active = true AND (k.next_eligible_at IS NULL OR k.next_eligible_at <= NOW())
		ORDER BY k.new_places_last_run DESC, COALESCE(k.last_scraped_at, '1970-01-01'::timestamptz) ASC, k.created_at ASC`)
}

// ListAllDue is the scheduler's fallback when no keyword is active (existing behaviour: run all).
func (r *KeywordRepository) ListAllDue(ctx context.Context, limit int) ([]*model.ScraperKeyword, error) {
	return r.queryKeywords(ctx, keywordSelect+`
		WHERE (k.next_eligible_at IS NULL OR k.next_eligible_at <= NOW())
		ORDER BY k.new_places_last_run DESC, COALESCE(k.last_scraped_at, '1970-01-01'::timestamptz) ASC, k.created_at ASC
		LIMIT $1`, limit)
}

// RestoreOrphanedProcessing fixes keywords stuck in "processing" with no queued or running job
// (for example after a crash before the queue existed). They go back to paginating or ready.
func (r *KeywordRepository) RestoreOrphanedProcessing(ctx context.Context) (int, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE scraper_keywords k
		SET status = CASE
		        WHEN k.next_page_token <> '' THEN 'paginating'
		        WHEN EXISTS (SELECT 1 FROM keyword_cells c WHERE c.keyword_id = k.id AND c.status = 'pending')
		         AND EXISTS (SELECT 1 FROM keyword_cells c WHERE c.keyword_id = k.id AND c.status <> 'pending') THEN 'paginating'
		        ELSE 'ready' END,
		    updated_at = NOW()
		WHERE k.status = 'processing'
		  AND NOT EXISTS (
		      SELECT 1 FROM scrape_jobs j
		      WHERE j.keyword_id = k.id AND j.status IN ('PENDING', 'PROCESSING')
		  )`)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// ---------------------------------------------------------------------------------------------
// Area cells

const cellColumns = `id, keyword_id, depth, low_lat, low_lng, high_lat, high_lng, status, next_page_token, pages_fetched, results`

func scanCell(row pgx.Row) (*model.KeywordCell, error) {
	c := &model.KeywordCell{}
	err := row.Scan(&c.ID, &c.KeywordID, &c.Depth, &c.LowLat, &c.LowLng, &c.HighLat, &c.HighLng, &c.Status, &c.NextPageToken, &c.PagesFetched, &c.Results)
	return c, err
}

// CountCells returns how many cells a keyword's current cycle has (any status).
func (r *KeywordRepository) CountCells(ctx context.Context, keywordID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM keyword_cells WHERE keyword_id = $1`, keywordID).Scan(&n)
	return n, err
}

// StartCycle replaces a keyword's cells with a single root cell covering its area.
func (r *KeywordRepository) StartCycle(ctx context.Context, keywordID uuid.UUID, area *model.Area) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM keyword_cells WHERE keyword_id = $1`, keywordID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO keyword_cells (keyword_id, depth, low_lat, low_lng, high_lat, high_lng)
		VALUES ($1, 0, $2, $3, $4, $5)`, keywordID, area.LowLat, area.LowLng, area.HighLat, area.HighLng); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// NextPendingCell returns the cell to search next: an in-progress one first, then shallow ones.
func (r *KeywordRepository) NextPendingCell(ctx context.Context, keywordID uuid.UUID) (*model.KeywordCell, error) {
	c, err := scanCell(r.db.QueryRow(ctx, `
		SELECT `+cellColumns+` FROM keyword_cells
		WHERE keyword_id = $1 AND status = 'pending'
		ORDER BY (pages_fetched > 0) DESC, depth, created_at, id
		LIMIT 1`, keywordID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

// SaveCellProgress stores a cell's page token and counts; status is pending or done.
func (r *KeywordRepository) SaveCellProgress(ctx context.Context, c *model.KeywordCell) error {
	_, err := r.db.Exec(ctx, `
		UPDATE keyword_cells
		SET status = $2, next_page_token = $3, pages_fetched = $4, results = $5, updated_at = NOW()
		WHERE id = $1`, c.ID, c.Status, c.NextPageToken, c.PagesFetched, c.Results)
	return err
}

// SplitCell marks a cell as split and adds its four quarters as pending cells.
func (r *KeywordRepository) SplitCell(ctx context.Context, c *model.KeywordCell) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		UPDATE keyword_cells SET status = 'split', next_page_token = '', pages_fetched = $2, results = $3, updated_at = NOW()
		WHERE id = $1`, c.ID, c.PagesFetched, c.Results); err != nil {
		return err
	}
	midLat := (c.LowLat + c.HighLat) / 2
	midLng := (c.LowLng + c.HighLng) / 2
	quarters := [4][4]float64{
		{c.LowLat, c.LowLng, midLat, midLng},
		{c.LowLat, midLng, midLat, c.HighLng},
		{midLat, c.LowLng, c.HighLat, midLng},
		{midLat, midLng, c.HighLat, c.HighLng},
	}
	for _, q := range quarters {
		if _, err := tx.Exec(ctx, `
			INSERT INTO keyword_cells (keyword_id, depth, low_lat, low_lng, high_lat, high_lng)
			VALUES ($1, $2, $3, $4, $5, $6)`, c.KeywordID, c.Depth+1, q[0], q[1], q[2], q[3]); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// CycleResults sums results over the finished cells of the current cycle.
func (r *KeywordRepository) CycleResults(ctx context.Context, keywordID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COALESCE(SUM(results), 0) FROM keyword_cells WHERE keyword_id = $1 AND status <> 'split'`, keywordID).Scan(&n)
	return n, err
}
