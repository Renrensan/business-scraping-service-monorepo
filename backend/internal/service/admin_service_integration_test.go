package service

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"

	"business-scraping-service/backend/internal/apperr"
	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"
	"business-scraping-service/backend/internal/utils"
)

// memSessions is an in-memory session.Store.
type memSessions struct {
	mu       sync.Mutex
	sessions map[string]model.SessionData
}

func (m *memSessions) CreateSession(ctx context.Context, d model.SessionData) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sessions == nil {
		m.sessions = map[string]model.SessionData{}
	}
	tok := uuid.NewString()
	d.Token = tok
	m.sessions[tok] = d
	return tok, nil
}
func (m *memSessions) GetSession(ctx context.Context, tok string) (*model.SessionData, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if d, ok := m.sessions[tok]; ok {
		return &d, nil
	}
	return nil, nil
}
func (m *memSessions) TouchSession(ctx context.Context, tok string) error { return nil }
func (m *memSessions) DeleteSession(ctx context.Context, tok string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, tok)
	return nil
}
func (m *memSessions) DeleteAllUserSessions(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, d := range m.sessions {
		if d.AdminID == id {
			delete(m.sessions, k)
		}
	}
	return nil
}

const testPassword = "test-password-123" // placeholder, not a real credential

func TestAdmins_MultiAccountRulesAndContactedBy(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := repository.InitDB(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	for _, q := range []string{`TRUNCATE business_leads, audit_log CASCADE`, `DELETE FROM admin_users`} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	adminRepo := repository.NewAdminRepository(pool)
	sessions := &memSessions{}
	admins := NewAdminService(adminRepo, sessions)
	auth := NewAuthService(adminRepo, sessions)

	ann, err := admins.Create(ctx, CreateAdminInput{Email: " Ann@Example.com ", Password: testPassword})
	if err != nil || ann.Username != "ann" || ann.Email != "ann@example.com" {
		t.Fatalf("create: %+v %v", ann, err)
	}
	bob, err := admins.Create(ctx, CreateAdminInput{Email: "bob@example.com", Password: testPassword})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admins.Create(ctx, CreateAdminInput{Email: "bob@example.com", Password: testPassword}); !errors.Is(err, apperr.ErrConflict) {
		t.Errorf("duplicate email: %v", err)
	}
	if _, err := admins.Create(ctx, CreateAdminInput{Email: "x@example.com", Password: "short"}); !errors.Is(err, apperr.ErrInvalid) {
		t.Errorf("short password: %v", err)
	}

	// Both can log in; last login is recorded; hashes are never listed.
	_, tokBob, err := auth.Login(ctx, "bob@example.com", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := auth.Login(ctx, "ann", testPassword); err != nil {
		t.Fatal(err)
	}
	list, _ := admins.List(ctx)
	if len(list) != 2 || list[0].LastLoginAt == nil || list[0].PasswordHash != "" {
		t.Errorf("list: %+v", list)
	}

	off := false
	if _, err := admins.Update(ctx, bob.ID, bob.ID, UpdateAdminInput{IsActive: &off}); !errors.Is(err, apperr.ErrInvalid) {
		t.Errorf("self-deactivate: %v", err)
	}
	// Deactivating someone signs them out at once and blocks login.
	if _, err := admins.Update(ctx, ann.ID, bob.ID, UpdateAdminInput{IsActive: &off}); err != nil {
		t.Fatal(err)
	}
	if s, _ := auth.ValidateSession(ctx, tokBob); s != nil {
		t.Error("deactivated admin's session still valid")
	}
	if _, _, err := auth.Login(ctx, "bob@example.com", testPassword); err == nil {
		t.Error("deactivated admin could log in")
	}
	if err := adminRepo.SetActive(ctx, ann.ID, false); !errors.Is(err, apperr.ErrConflict) {
		t.Errorf("last active admin: %v", err)
	}

	// Upsert reactivates and resets an existing admin, or creates a new one.
	newPw := "another-test-pw-456"
	a, created, err := admins.Upsert(ctx, CreateAdminInput{Email: "bob@example.com", Password: newPw})
	if err != nil || created || !a.IsActive || !utils.CheckPasswordHash(newPw, mustHash(t, adminRepo, a.ID)) {
		t.Errorf("upsert existing: %+v created=%v err=%v", a, created, err)
	}
	if _, created, err := admins.Upsert(ctx, CreateAdminInput{Email: "carol@example.com", Password: newPw}); err != nil || !created {
		t.Errorf("upsert new: created=%v err=%v", created, err)
	}

	// contacted_by: the first admin to mark a lead contacted is kept; unmarking clears it.
	leads := repository.NewBusinessRepository(pool)
	l := &model.BusinessLead{PlaceID: "p1", Name: "Alaya", Address: "x"}
	if _, err := leads.Upsert(ctx, l); err != nil {
		t.Fatal(err)
	}
	if err := leads.SetContacted(ctx, l.ID, true, &ann.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := leads.SetLeadStatus(ctx, l.ID, model.LeadStatus("hot"), &bob.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := leads.GetByID(ctx, l.ID)
	if got.ContactedBy == nil || *got.ContactedBy != ann.ID || got.ContactedByName != "ann" {
		t.Errorf("contacted_by after status change: %v %q", got.ContactedBy, got.ContactedByName)
	}
	page, _ := leads.List(ctx, model.BusinessFilter{Page: 1, PageSize: 5})
	if page.Items[0].ContactedByName != "ann" {
		t.Errorf("list contacted_by_name %q", page.Items[0].ContactedByName)
	}
	_ = leads.SetContacted(ctx, l.ID, false, &bob.ID)
	got, _ = leads.GetByID(ctx, l.ID)
	if got.ContactedBy != nil || got.ContactedByName != "" {
		t.Errorf("unmark should clear contacted_by: %v", got.ContactedBy)
	}
	_, _ = leads.SetLeadStatus(ctx, l.ID, model.LeadStatus("warm"), &bob.ID)
	got, _ = leads.GetByID(ctx, l.ID)
	if got.ContactedByName != "bob" {
		t.Errorf("status on uncontacted lead: %q", got.ContactedByName)
	}
}

func mustHash(t *testing.T, repo *repository.AdminRepository, id uuid.UUID) string {
	t.Helper()
	a, err := repo.GetByID(context.Background(), id)
	if err != nil || a == nil {
		t.Fatal(err)
	}
	return a.PasswordHash
}
