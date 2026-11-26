package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"business-scraping-service/backend/internal/model"
)

// AuditStore saves audit entries.
type AuditStore interface {
	Insert(ctx context.Context, e *model.AuditEntry) error
}

// Recorder writes audit entries; also used directly for logins, which happen before a session exists.
type Recorder struct {
	store AuditStore
}

func NewRecorder(store AuditStore) *Recorder { return &Recorder{store: store} }

// Record saves an entry with a fresh context, so it is kept even if the request was cancelled.
// A failure is logged, never shown to the admin: the action itself already happened.
func (rec *Recorder) Record(r *http.Request, e *model.AuditEntry) {
	if rec == nil || rec.store == nil {
		return
	}
	if e.IP == "" {
		e.IP = clientIP(r)
	}
	if e.RequestID == "" {
		e.RequestID, _ = r.Context().Value(RequestIDKey).(string)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rec.store.Insert(ctx, e); err != nil {
		slog.Error("failed to write audit entry", "action", e.Action, "admin", e.AdminName, "error", err)
	}
}

func clientIP(r *http.Request) string {
	if i := strings.LastIndex(r.RemoteAddr, ":"); i > 0 && !strings.HasSuffix(r.RemoteAddr, "]") {
		return strings.Trim(r.RemoteAddr[:i], "[]")
	}
	return strings.Trim(r.RemoteAddr, "[]")
}

// apiPrefixes are the prefixes the router mounts the API under (see api.CreateRouter). In
// production nginx strips "/api", so requests arrive as /v1/...; longest prefix first.
var apiPrefixes = []string{"/api/v1", "/v1", "/api"}

// stripAPIPrefix turns "/v1/businesses/{id}" (or /api/v1/..., /api/...) into "/businesses/{id}".
func stripAPIPrefix(p string) string {
	for _, prefix := range apiPrefixes {
		if strings.HasPrefix(p, prefix+"/") {
			return strings.TrimPrefix(p, prefix)
		}
	}
	return p
}

// auditedGET lists read requests that are still worth auditing: they copy lead data out.
var auditedGET = map[string]bool{
	"/export/csv":         true,
	"/export/wa-contacts": true,
}

// Audit records every change an admin makes through the API (and exports): who, what, which
// record, the outcome, and the request body with secrets removed. It must run after
// AuthMiddleware so the session is known.
func (rec *Recorder) Audit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isRead := r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions
		if isRead && !auditedGET[stripAPIPrefix(r.URL.Path)] {
			next.ServeHTTP(w, r)
			return
		}

		var body []byte
		if r.Body != nil {
			body, _ = io.ReadAll(io.LimitReader(r.Body, 64<<10))
			r.Body = io.NopCloser(bytes.NewReader(body))
		}
		cw := &captureWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(cw, r)

		route := stripAPIPrefix(chi.RouteContext(r.Context()).RoutePattern())
		e := describe(r.Method, route, body, cw.body.Bytes())
		e.Method, e.Route, e.Status = r.Method, route, cw.status
		if e.EntityID == "" {
			e.EntityID = chi.URLParam(r, "id")
		}
		if s := GetSessionFromContext(r.Context()); s != nil {
			id := s.AdminID
			e.AdminID, e.AdminName = &id, s.Username
		}
		e.Details = buildDetails(body, r.URL.RawQuery, cw.status, cw.body.Bytes())
		rec.Record(r, e)
	})
}

// captureWriter keeps the status and the start of the response (to read new ids and error messages).
type captureWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	body        bytes.Buffer
}

func (c *captureWriter) WriteHeader(code int) {
	if !c.wroteHeader {
		c.status, c.wroteHeader = code, true
	}
	c.ResponseWriter.WriteHeader(code)
}

func (c *captureWriter) Write(p []byte) (int, error) {
	if c.body.Len() < 8<<10 {
		c.body.Write(p[:min(len(p), 8<<10-c.body.Len())])
	}
	return c.ResponseWriter.Write(p)
}

// Flush keeps streaming responses (CSV downloads) working through the wrapper.
func (c *captureWriter) Flush() {
	if f, ok := c.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

type routeInfo struct {
	action, summary, entity string
}

// routes maps "METHOD /pattern" to what the action means. Unknown routes are still recorded.
var routes = map[string]routeInfo{
	"POST /auth/logout":                            {"auth.logout", "Logged out", ""},
	"POST /auth/change-password":                   {"auth.change_password", "Changed own password", "admin"},
	"POST /jobs":                                   {"job.create", "Started a manual scrape", ""},
	"POST /keywords":                               {"keyword.create", "Added keywords", "keyword"},
	"PUT /keywords/{id}":                           {"keyword.update", "Updated keyword", "keyword"},
	"PATCH /keywords/{id}":                         {"keyword.update", "Updated keyword", "keyword"},
	"POST /keywords/{id}/scrape":                   {"keyword.scrape", "Started keyword scrape", "keyword"},
	"POST /keywords/{id}/trigger":                  {"keyword.scrape", "Started keyword scrape", "keyword"},
	"POST /keywords/{id}/reset":                    {"keyword.reset", "Reset keyword to the start", "keyword"},
	"DELETE /keywords/{id}":                        {"keyword.delete", "Deleted keyword", "keyword"},
	"POST /areas":                                  {"area.create", "Added area", "area"},
	"DELETE /areas/{id}":                           {"area.delete", "Deleted area", "area"},
	"PATCH /businesses/{id}/contacted":             {"lead.contacted", "Changed contacted", "lead"},
	"PATCH /businesses/{id}/status":                {"lead.status", "Changed lead status", "lead"},
	"PATCH /businesses/{id}/grouping":              {"lead.grouping", "Changed lead group", "lead"},
	"POST /businesses/{id}/contacts":               {"lead.contact_add", "Added contact person", "lead"},
	"PUT /businesses/{id}/contacts/{contactID}":    {"lead.contact_update", "Updated contact person", "lead"},
	"DELETE /businesses/{id}/contacts/{contactID}": {"lead.contact_delete", "Removed contact person", "lead"},
	"PATCH /businesses/{id}/scores/{profileID}":    {"lead.tier", "Changed lead tier / note", "lead"},
	"DELETE /businesses/{id}":                      {"lead.delete", "Deleted lead", "lead"},
	"POST /scoring/profiles":                       {"scoring.create", "Created scoring profile", "scoring_profile"},
	"PUT /scoring/profiles/{id}":                   {"scoring.update", "Updated scoring profile", "scoring_profile"},
	"DELETE /scoring/profiles/{id}":                {"scoring.delete", "Deleted scoring profile", "scoring_profile"},
	"POST /scoring/rescore":                        {"scoring.rescore", "Rescored all leads", ""},
	"POST /system/api-key":                         {"system.api_key", "Changed the Google Places API key", ""},
	"POST /system/places-budget":                   {"system.places_budget", "Changed the daily request limit", ""},
	"POST /system/schedule":                        {"system.schedule", "Changed the auto-scrape schedule", ""},
	"POST /system/trigger-scrape":                  {"system.trigger_scrape", "Started a scrape run", ""},
	"POST /admins":                                 {"admin.create", "Added admin", "admin"},
	"PATCH /admins/{id}":                           {"admin.update", "Changed admin", "admin"},
	"GET /export/csv":                              {"export.csv", "Exported leads (CSV)", ""},
	"GET /export/wa-contacts":                      {"export.whatsapp", "Exported WhatsApp contacts", ""},
}

// describe turns a request into an entry with a readable summary.
func describe(method, route string, reqBody, respBody []byte) *model.AuditEntry {
	info, ok := routes[method+" "+route]
	if !ok {
		info = routeInfo{"api." + strings.ToLower(method), method + " " + route, ""}
	}
	e := &model.AuditEntry{Action: info.action, Summary: info.summary, EntityType: info.entity}

	var in map[string]any
	_ = json.Unmarshal(reqBody, &in) // non-JSON bodies just get the generic summary
	str := func(k string) string { s, _ := in[k].(string); return strings.TrimSpace(s) }

	switch e.Action {
	case "lead.contacted":
		if v, ok := in["contacted"].(bool); ok && !v {
			e.Summary = "Unmarked lead as contacted"
		} else {
			e.Summary = "Marked lead as contacted"
		}
	case "lead.status":
		if s := str("status"); s != "" {
			e.Summary = "Set lead status to " + strings.ToUpper(s)
		}
	case "lead.contact_add", "lead.contact_update":
		if n := str("name"); n != "" {
			e.Summary += ": " + n
		}
	case "lead.tier":
		if t := str("tier_override"); t != "" {
			e.Summary = "Set tier to " + strings.ToUpper(t) + " by hand"
		}
	case "admin.create":
		if n := str("email"); n != "" {
			e.Summary += ": " + n
		}
	case "admin.update":
		switch v, ok := in["is_active"].(bool); {
		case ok && v:
			e.Summary = "Reactivated admin"
		case ok && !v:
			e.Summary = "Deactivated admin"
		case str("password") != "":
			e.Summary = "Reset admin password"
		}
	case "system.places_budget":
		if v, ok := in["daily_budget"].(float64); ok {
			e.Summary = fmt.Sprintf("Set the daily request limit to %d", int(v))
		}
	}

	// Creates don't have the new id in the URL: take it from the response.
	if e.EntityType != "" && (strings.HasSuffix(e.Action, ".create")) {
		var out struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if json.Unmarshal(respBody, &out) == nil {
			e.EntityID = out.Data.ID
		}
	}
	return e
}

var secretKeys = []string{"password", "api_key", "apikey", "token", "secret"}

// buildDetails keeps the request body (secrets replaced), the query string, and the error
// message of a failed request.
func buildDetails(reqBody []byte, query string, status int, respBody []byte) json.RawMessage {
	d := map[string]any{}
	var in any
	if len(reqBody) > 0 && json.Unmarshal(reqBody, &in) == nil {
		d["request"] = redact(in)
	}
	if query != "" {
		d["query"] = query
	}
	if status >= 400 {
		var out struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(respBody, &out) == nil && out.Error != "" {
			d["error"] = out.Error
		}
	}
	b, err := json.Marshal(d)
	if err != nil {
		return json.RawMessage("{}")
	}
	return b
}

func redact(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			secret := false
			for _, s := range secretKeys {
				if strings.Contains(strings.ToLower(k), s) {
					secret = true
					break
				}
			}
			if secret {
				out[k] = "[redacted]"
			} else {
				out[k] = redact(val)
			}
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = redact(val)
		}
		return out
	}
	return v
}
