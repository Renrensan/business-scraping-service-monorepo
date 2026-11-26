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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"business-scraping-service/backend/internal/model"
)

var (
	ErrProfileExists    = errors.New("a scoring profile with this name already exists")
	ErrProfileNotFound  = errors.New("scoring profile not found")
	ErrScoreNotFound    = errors.New("this lead has no score for that profile (the profile does not apply to it)")
	ErrInvalidTier      = errors.New("tier must be hot, good, low or skip")
	ErrInvalidGroupType = errors.New("invalid group type")
)

type ScoringRepository struct {
	db *pgxpool.Pool
}

func NewScoringRepository(db *pgxpool.Pool) *ScoringRepository {
	return &ScoringRepository{db: db}
}

// ---------------------------------------------------------------------------------------------
// Profiles

const profileSelect = `
	SELECT p.id, p.name, p.description, p.applies_to_types, p.config, p.is_active,
	       (SELECT COUNT(*) FROM lead_scores s WHERE s.profile_id = p.id), p.created_at, p.updated_at
	FROM scoring_profiles p`

func scanProfile(row pgx.Row) (*model.ScoringProfile, error) {
	p := &model.ScoringProfile{}
	var applies []byte
	if err := row.Scan(&p.ID, &p.Name, &p.Description, &applies, &p.Config, &p.IsActive, &p.ScoredLeads, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(applies, &p.AppliesToTypes)
	if p.AppliesToTypes == nil {
		p.AppliesToTypes = []string{}
	}
	return p, nil
}

func (r *ScoringRepository) ListProfiles(ctx context.Context, activeOnly bool) ([]*model.ScoringProfile, error) {
	q := profileSelect
	if activeOnly {
		q += ` WHERE p.is_active`
	}
	rows, err := r.db.Query(ctx, q+` ORDER BY p.created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.ScoringProfile
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *ScoringRepository) GetProfile(ctx context.Context, id uuid.UUID) (*model.ScoringProfile, error) {
	p, err := scanProfile(r.db.QueryRow(ctx, profileSelect+` WHERE p.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProfileNotFound
	}
	return p, err
}

// GetProfileByName finds a profile by name, case-insensitive.
func (r *ScoringRepository) GetProfileByName(ctx context.Context, name string) (*model.ScoringProfile, error) {
	p, err := scanProfile(r.db.QueryRow(ctx, profileSelect+` WHERE LOWER(p.name) = LOWER($1)`, strings.TrimSpace(name)))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProfileNotFound
	}
	return p, err
}

func profileWriteErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrProfileExists
	}
	return err
}

func (r *ScoringRepository) CreateProfile(ctx context.Context, p *model.ScoringProfile) error {
	applies, _ := json.Marshal(p.AppliesToTypes)
	err := r.db.QueryRow(ctx, `
		INSERT INTO scoring_profiles (name, description, applies_to_types, config, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`,
		strings.TrimSpace(p.Name), p.Description, applies, []byte(p.Config), p.IsActive).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	return profileWriteErr(err)
}

func (r *ScoringRepository) UpdateProfile(ctx context.Context, p *model.ScoringProfile) error {
	applies, _ := json.Marshal(p.AppliesToTypes)
	tag, err := r.db.Exec(ctx, `
		UPDATE scoring_profiles
		SET name = $2, description = $3, applies_to_types = $4, config = $5, is_active = $6, updated_at = NOW()
		WHERE id = $1`, p.ID, strings.TrimSpace(p.Name), p.Description, applies, []byte(p.Config), p.IsActive)
	if err != nil {
		return profileWriteErr(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrProfileNotFound
	}
	return nil
}

func (r *ScoringRepository) DeleteProfile(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM scoring_profiles WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProfileNotFound
	}
	return nil
}

// ---------------------------------------------------------------------------------------------
// Leads to score

// ScoringLead is the slice of a lead that scoring reads.
type ScoringLead struct {
	ID                 uuid.UUID
	Name               string
	Category           string
	Types              []string
	Address            string
	Latitude           float64
	Longitude          float64
	Rating             float64
	ReviewCount        int
	Website            string
	PrimaryPhone       string
	Emails             []string
	FacebookURL        string
	InstagramURL       string
	TikTokURL          string
	LinkedInURL        string
	TwitterURL         string
	YouTubeURL         string
	GroupType          *string
	GroupName          *string
	SuggestedGroupType string
	SuggestedGroupName string
}

// LeadsForScoring loads the given leads, or all leads when ids is nil.
func (r *ScoringRepository) LeadsForScoring(ctx context.Context, ids []uuid.UUID) ([]*ScoringLead, error) {
	q := `
		SELECT id, COALESCE(name, ''), COALESCE(category, ''), COALESCE(subcategories, '[]'::jsonb), COALESCE(address, ''),
		       COALESCE(latitude, 0), COALESCE(longitude, 0), COALESCE(rating, 0), COALESCE(review_count, 0),
		       website, primary_phone, COALESCE(emails, '[]'::jsonb), facebook_url, instagram_url, tiktok_url, linkedin_url,
		       twitter_url, youtube_url, group_type, group_name, suggested_group_type, suggested_group_name
		FROM business_leads`
	var args []any
	if ids != nil {
		if len(ids) == 0 {
			return nil, nil
		}
		q += ` WHERE id = ANY($1)`
		args = append(args, ids)
	}
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ScoringLead
	for rows.Next() {
		l := &ScoringLead{}
		var types, emails []byte
		var website, phone, fb, ig, tt, li, tw, yt *string
		if err := rows.Scan(&l.ID, &l.Name, &l.Category, &types, &l.Address, &l.Latitude, &l.Longitude, &l.Rating, &l.ReviewCount,
			&website, &phone, &emails, &fb, &ig, &tt, &li, &tw, &yt,
			&l.GroupType, &l.GroupName, &l.SuggestedGroupType, &l.SuggestedGroupName); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(types, &l.Types)
		_ = json.Unmarshal(emails, &l.Emails)
		deref := func(p *string) string {
			if p == nil {
				return ""
			}
			return *p
		}
		l.Website, l.PrimaryPhone = deref(website), deref(phone)
		l.FacebookURL, l.InstagramURL, l.TikTokURL = deref(fb), deref(ig), deref(tt)
		l.LinkedInURL, l.TwitterURL, l.YouTubeURL = deref(li), deref(tw), deref(yt)
		out = append(out, l)
	}
	return out, rows.Err()
}

// ScoreRow is one computed score to store.
type ScoreRow struct {
	LeadID       uuid.UUID
	ProfileID    uuid.UUID
	Score        int
	RawScore     float64
	ComputedTier string
	AreaName     string
	Breakdown    []byte
}

// SaveScores writes computed scores and suggestions in one transaction. For the given leads (all
// leads when leadIDs is nil), scores for profiles that no longer apply are removed. Manual tier
// overrides and notes are kept.
func (r *ScoringRepository) SaveScores(ctx context.Context, leadIDs []uuid.UUID, rows []ScoreRow, suggestions map[uuid.UUID][2]string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Remove scores that are no longer computed (profile deactivated or no longer applies).
	keep := make([]string, 0, len(rows))
	for _, s := range rows {
		keep = append(keep, s.LeadID.String()+"|"+s.ProfileID.String())
	}
	if leadIDs == nil {
		_, err = tx.Exec(ctx, `DELETE FROM lead_scores WHERE NOT ((lead_id::text || '|' || profile_id::text) = ANY($1))`, keep)
	} else {
		_, err = tx.Exec(ctx, `DELETE FROM lead_scores WHERE lead_id = ANY($1) AND NOT ((lead_id::text || '|' || profile_id::text) = ANY($2))`, leadIDs, keep)
	}
	if err != nil {
		return fmt.Errorf("failed to clear old scores: %w", err)
	}

	batch := &pgx.Batch{}
	now := time.Now().UTC()
	for _, s := range rows {
		batch.Queue(`
			INSERT INTO lead_scores (lead_id, profile_id, score, raw_score, computed_tier, area_name, breakdown, computed_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (lead_id, profile_id) DO UPDATE SET
				score = EXCLUDED.score, raw_score = EXCLUDED.raw_score, computed_tier = EXCLUDED.computed_tier,
				area_name = EXCLUDED.area_name, breakdown = EXCLUDED.breakdown, computed_at = EXCLUDED.computed_at`,
			s.LeadID, s.ProfileID, s.Score, s.RawScore, s.ComputedTier, s.AreaName, s.Breakdown, now)
	}
	for id, sg := range suggestions {
		batch.Queue(`
			UPDATE business_leads SET suggested_group_type = $2, suggested_group_name = $3
			WHERE id = $1 AND (suggested_group_type <> $2 OR suggested_group_name <> $3)`, id, sg[0], sg[1])
	}
	if batch.Len() > 0 {
		if err := tx.SendBatch(ctx, batch).Close(); err != nil {
			return fmt.Errorf("failed to save scores: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// ---------------------------------------------------------------------------------------------
// Manual edits

// SetLeadGrouping sets a lead's group type and group name by hand (nil clears back to the suggestion).
func (r *ScoringRepository) SetLeadGrouping(ctx context.Context, leadID uuid.UUID, groupType, groupName *string) error {
	if groupType != nil && *groupType == "" {
		groupType = nil
	}
	if groupName != nil {
		trimmed := strings.TrimSpace(*groupName)
		groupName = &trimmed
		if trimmed == "" {
			groupName = nil
		}
	}
	tag, err := r.db.Exec(ctx, `UPDATE business_leads SET group_type = $2, group_name = $3, updated_at = NOW() WHERE id = $1`, leadID, groupType, groupName)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// SetScoreOverride sets (or clears, with nil) a lead's tier override and note for a profile.
func (r *ScoringRepository) SetScoreOverride(ctx context.Context, leadID, profileID uuid.UUID, tierOverride *string, note string) error {
	if tierOverride != nil && *tierOverride == "" {
		tierOverride = nil
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE lead_scores SET tier_override = $3, note = $4 WHERE lead_id = $1 AND profile_id = $2`,
		leadID, profileID, tierOverride, strings.TrimSpace(note))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23514" {
			return ErrInvalidTier
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrScoreNotFound
	}
	return nil
}

// ScoresForLead returns a lead's scores under every profile that applies to it.
func (r *ScoringRepository) ScoresForLead(ctx context.Context, leadID uuid.UUID) ([]model.LeadScore, error) {
	rows, err := r.db.Query(ctx, `
		SELECT s.profile_id, p.name, s.score, s.raw_score, s.tier, s.computed_tier, s.tier_override,
		       s.note, s.area_name, s.breakdown, s.computed_at
		FROM lead_scores s JOIN scoring_profiles p ON p.id = s.profile_id
		WHERE s.lead_id = $1
		ORDER BY p.created_at`, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.LeadScore{}
	for rows.Next() {
		var s model.LeadScore
		if err := rows.Scan(&s.ProfileID, &s.ProfileName, &s.Score, &s.RawScore, &s.Tier, &s.ComputedTier, &s.TierOverride,
			&s.Note, &s.AreaName, &s.Breakdown, &s.ComputedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
