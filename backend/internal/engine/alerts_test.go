package engine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestAlerter_SendsToEvolutionAndThrottles(t *testing.T) {
	var mu sync.Mutex
	var got []map[string]string
	var gotKey, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]string
		_ = json.NewDecoder(r.Body).Decode(&b)
		mu.Lock()
		got = append(got, b)
		gotKey, gotPath = r.Header.Get("apikey"), r.URL.Path
		mu.Unlock()
		w.WriteHeader(201)
	}))
	defer srv.Close()

	a := NewAlerter(srv.URL+"/", "secret", "main", "62811")
	now := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)
	a.now = func() time.Time { return now }
	if !a.Enabled() {
		t.Fatal("should be enabled")
	}

	a.Notify("quota", "quota exhausted")
	a.Notify("quota", "quota exhausted again") // throttled
	a.Notify("job_failed", "x failed")         // other kind: sent
	now = now.Add(61 * time.Minute)
	a.Notify("quota", "next day") // window passed: sent

	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		n := len(got)
		mu.Unlock()
		if n >= 3 || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 3 {
		t.Fatalf("sent %d alerts, want 3: %v", len(got), got)
	}
	if gotKey != "secret" || gotPath != "/message/sendText/main" || got[0]["number"] != "62811" || got[0]["text"] == "" {
		t.Errorf("request: key=%q path=%q body=%v", gotKey, gotPath, got[0])
	}
}

func TestAlerter_LogOnlyWhenNotConfigured(t *testing.T) {
	a := NewAlerter("", "", "", "")
	if a.Enabled() {
		t.Error("should be log-only")
	}
	a.Notify("quota", "x") // must not panic
	var nilAlerter *Alerter
	nilAlerter.Notify("quota", "x")
	_ = context.Background()
}
