package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"business-scraping-service/backend/internal/apperr"
	"business-scraping-service/backend/internal/model"
)

var (
	ErrContactNotFound = apperr.NotFound("contact not found")
	ErrLeadNotFound    = apperr.NotFound("lead not found")
)

// ContactRepository stores the people admins record for a lead.
type ContactRepository struct {
	db *pgxpool.Pool
}

func NewContactRepository(db *pgxpool.Pool) *ContactRepository {
	return &ContactRepository{db: db}
}

const contactColumns = `id, lead_id, name, role, phone, email, note, created_at, updated_at`

func scanContact(row pgx.Row) (*model.LeadContact, error) {
	c := &model.LeadContact{}
	err := row.Scan(&c.ID, &c.LeadID, &c.Name, &c.Role, &c.Phone, &c.Email, &c.Note, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

// List returns a lead's contacts, oldest first (the first one added is the main contact).
func (r *ContactRepository) List(ctx context.Context, leadID uuid.UUID) ([]model.LeadContact, error) {
	rows, err := r.db.Query(ctx, `SELECT `+contactColumns+` FROM lead_contacts WHERE lead_id = $1 ORDER BY created_at, id`, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.LeadContact{}
	for rows.Next() {
		c, err := scanContact(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// Create adds a contact to a lead. It returns ErrLeadNotFound when the lead doesn't exist.
func (r *ContactRepository) Create(ctx context.Context, leadID uuid.UUID, in model.LeadContactInput) (*model.LeadContact, error) {
	c, err := scanContact(r.db.QueryRow(ctx, `
		INSERT INTO lead_contacts (lead_id, name, role, phone, email, note)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+contactColumns, leadID, in.Name, in.Role, in.Phone, in.Email, in.Note))
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" { // foreign key: no such lead
		return nil, ErrLeadNotFound
	}
	return c, err
}

// Update changes a contact of the given lead.
func (r *ContactRepository) Update(ctx context.Context, leadID, contactID uuid.UUID, in model.LeadContactInput) (*model.LeadContact, error) {
	c, err := scanContact(r.db.QueryRow(ctx, `
		UPDATE lead_contacts
		SET name = $3, role = $4, phone = $5, email = $6, note = $7, updated_at = NOW()
		WHERE id = $2 AND lead_id = $1
		RETURNING `+contactColumns, leadID, contactID, in.Name, in.Role, in.Phone, in.Email, in.Note))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrContactNotFound
	}
	return c, err
}

// Delete removes a contact of the given lead.
func (r *ContactRepository) Delete(ctx context.Context, leadID, contactID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM lead_contacts WHERE id = $2 AND lead_id = $1`, leadID, contactID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrContactNotFound
	}
	return nil
}
