package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"business-scraping-service/backend/internal/model"
)

// AuditRepository stores and lists admin actions.
type AuditRepository struct {
	db *pgxpool.Pool
}

func NewAuditRepository(db *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Insert(ctx context.Context, e *model.AuditEntry) error {
	details := []byte(e.Details)
	if len(details) == 0 {
		details = []byte("{}")
	}
	return r.db.QueryRow(ctx, `
		INSERT INTO audit_log (admin_id, admin_name, action, summary, entity_type, entity_id, method, route, status, details, ip, request_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at`,
		e.AdminID, e.AdminName, e.Action, e.Summary, e.EntityType, e.EntityID, e.Method, e.Route, e.Status, details, e.IP, e.RequestID,
	).Scan(&e.ID, &e.CreatedAt)
}

// List returns entries matching the filter, newest first, with entity names resolved where the
// entity still exists (leads, keywords, areas, scoring profiles, admins).
func (r *AuditRepository) List(ctx context.Context, f model.AuditFilter) (*model.AuditListResponse, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 200 {
		f.PageSize = 50
	}
	where := []string{"1=1"}
	var args []any
	add := func(clause string, v any) {
		args = append(args, v)
		where = append(where, fmt.Sprintf(clause, len(args)))
	}
	if f.AdminID != nil {
		add("a.admin_id = $%d", *f.AdminID)
	}
	if f.EntityType != "" {
		add("a.entity_type = $%d", f.EntityType)
	}
	if f.EntityID != "" {
		add("a.entity_id = $%d", f.EntityID)
	}
	if f.Action != "" {
		add("a.action = $%d", f.Action)
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		add("(a.summary ILIKE $%[1]d OR a.action ILIKE $%[1]d OR a.admin_name ILIKE $%[1]d OR a.details::text ILIKE $%[1]d)", "%"+s+"%")
	}
	if f.FailedOnly {
		where = append(where, "a.status >= 400")
	}
	whereSQL := strings.Join(where, " AND ")

	var total int64
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM audit_log a WHERE "+whereSQL, args...).Scan(&total); err != nil {
		return nil, err
	}

	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT a.id, a.created_at, a.admin_id, a.admin_name, a.action, a.summary, a.entity_type, a.entity_id,
		       a.method, a.route, a.status, a.details, a.ip, a.request_id,
		       COALESCE(CASE a.entity_type
		           WHEN 'lead' THEN (SELECT name FROM business_leads WHERE id::text = a.entity_id)
		           WHEN 'keyword' THEN (SELECT keyword FROM scraper_keywords WHERE id::text = a.entity_id)
		           WHEN 'area' THEN (SELECT name FROM areas WHERE id::text = a.entity_id)
		           WHEN 'scoring_profile' THEN (SELECT name FROM scoring_profiles WHERE id::text = a.entity_id)
		           WHEN 'admin' THEN (SELECT username FROM admin_users WHERE id::text = a.entity_id)
		       END, '')
		FROM audit_log a
		WHERE %s
		ORDER BY a.created_at DESC, a.id DESC
		LIMIT $%d OFFSET $%d`, whereSQL, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.AuditEntry{}
	for rows.Next() {
		var e model.AuditEntry
		if err := rows.Scan(&e.ID, &e.CreatedAt, &e.AdminID, &e.AdminName, &e.Action, &e.Summary, &e.EntityType, &e.EntityID,
			&e.Method, &e.Route, &e.Status, &e.Details, &e.IP, &e.RequestID, &e.EntityName); err != nil {
			return nil, err
		}
		items = append(items, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	pages := int((total + int64(f.PageSize) - 1) / int64(f.PageSize))
	if pages == 0 {
		pages = 1
	}
	return &model.AuditListResponse{Items: items, TotalCount: total, Page: f.Page, PageSize: f.PageSize, TotalPages: pages}, nil
}
