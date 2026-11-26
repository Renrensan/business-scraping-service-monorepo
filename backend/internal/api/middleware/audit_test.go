package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"business-scraping-service/backend/internal/model"
)

type memAudit struct {
	mu      sync.Mutex
	entries []model.AuditEntry
}

func (m *memAudit) Insert(ctx context.Context, e *model.AuditEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, *e)
	return nil
}

func auditRouter(store *memAudit, admin uuid.UUID) http.Handler {
	rec := NewRecorder(store)
	r := chi.NewRouter()
	r.Route("/api/v1", func(api chi.Router) {
		api.Group(func(p chi.Router) {
			p.Use(func(next http.Handler) http.Handler { // stands in for AuthMiddleware
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					s := &model.SessionData{AdminID: admin, Username: "adiztraa"}
					next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), SessionContextKey, s)))
				})
			})
			p.Use(rec.Audit)
			ok := func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"success":true}`)) }
			p.Patch("/businesses/{id}/contacted", ok)
			p.Post("/businesses/{id}/contacts", ok)
			p.Post("/system/api-key", ok)
			p.Get("/businesses", ok)
			p.Get("/export/csv", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("Name,Phone\n")) })
			p.Post("/admins", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"success":true,"data":{"id":"11111111-1111-1111-1111-111111111111"}}`))
			})
			p.Patch("/admins/{id}", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"success":false,"error":"you can't deactivate your own account"}`))
			})
		})
	})
	return r
}

func TestAudit(t *testing.T) {
	admin := uuid.New()
	lead := uuid.New().String()
	store := &memAudit{}
	h := auditRouter(store, admin)
	do := func(method, path, body string) {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.RemoteAddr = "103.189.63.145:4000"
		h.ServeHTTP(httptest.NewRecorder(), req)
	}

	do("PATCH", "/api/v1/businesses/"+lead+"/contacted", `{"contacted":false}`)
	do("POST", "/api/v1/businesses/"+lead+"/contacts", `{"name":"Made","phone":"0812"}`)
	do("POST", "/api/v1/system/api-key", `{"api_key":"AIzaSecret123"}`)
	do("GET", "/api/v1/businesses?page=1", "")
	do("GET", "/api/v1/export/csv?score_profile=GuestChatPro&tiers=hot", "")
	do("POST", "/api/v1/admins", `{"email":"brewpending@gmail.com","password":"hunter22hunter"}`)
	do("PATCH", "/api/v1/admins/"+admin.String(), `{"is_active":false}`)

	if len(store.entries) != 6 {
		t.Fatalf("recorded %d entries, want 6 (plain GET is not audited): %+v", len(store.entries), store.entries)
	}
	byAction := map[string]model.AuditEntry{}
	for _, e := range store.entries {
		byAction[e.Action] = e
		if e.AdminID == nil || *e.AdminID != admin || e.AdminName != "adiztraa" || e.IP != "103.189.63.145" {
			t.Errorf("%s: who/where not recorded: %+v", e.Action, e)
		}
	}

	if e := byAction["lead.contacted"]; e.Summary != "Unmarked lead as contacted" || e.EntityType != "lead" || e.EntityID != lead || e.Route != "/businesses/{id}/contacted" {
		t.Errorf("contacted: %+v", e)
	}
	if e := byAction["lead.contact_add"]; e.Summary != "Added contact person: Made" || e.EntityID != lead {
		t.Errorf("contact add: %+v", e)
	}
	if e := byAction["system.api_key"]; strings.Contains(string(e.Details), "AIzaSecret123") || !strings.Contains(string(e.Details), "[redacted]") {
		t.Errorf("api key not redacted: %s", e.Details)
	}
	if e := byAction["export.csv"]; !strings.Contains(string(e.Details), "tiers=hot") {
		t.Errorf("export query not kept: %s", e.Details)
	}
	e := byAction["admin.create"]
	if e.EntityID != "11111111-1111-1111-1111-111111111111" || strings.Contains(string(e.Details), "hunter22") || e.Summary != "Added admin: brewpending@gmail.com" {
		t.Errorf("admin create: %+v %s", e, e.Details)
	}
	e = byAction["admin.update"]
	var d map[string]any
	_ = json.Unmarshal(e.Details, &d)
	if e.Status != 400 || e.Summary != "Deactivated admin" || d["error"] != "you can't deactivate your own account" {
		t.Errorf("failed admin update: %+v %s", e, e.Details)
	}
}
