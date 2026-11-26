package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Alerter sends short operational alerts (quota exhausted, failing jobs, keywords with no results).
// Each alert kind is sent at most once per throttle window; everything is also logged.
type Alerter struct {
	send     func(ctx context.Context, text string) error // nil: log only
	throttle time.Duration

	mu   sync.Mutex
	last map[string]time.Time
	now  func() time.Time
}

// NewAlerter returns an alerter that posts to an Evolution API WhatsApp instance when all of
// baseURL, apiKey, instance and number are set, and only logs otherwise.
func NewAlerter(baseURL, apiKey, instance, number string) *Alerter {
	a := &Alerter{throttle: time.Hour, last: map[string]time.Time{}, now: time.Now}
	if baseURL != "" && apiKey != "" && instance != "" && number != "" {
		a.send = evolutionSender(strings.TrimRight(baseURL, "/"), apiKey, instance, number)
	}
	return a
}

// Enabled reports whether alerts are actually delivered (not just logged).
func (a *Alerter) Enabled() bool { return a != nil && a.send != nil }

// Notify sends text unless an alert of the same kind was sent within the throttle window.
func (a *Alerter) Notify(kind, text string) {
	if a == nil {
		return
	}
	slog.Warn("ALERT", "kind", kind, "text", text)
	if a.send == nil {
		return
	}
	a.mu.Lock()
	if t, ok := a.last[kind]; ok && a.now().Sub(t) < a.throttle {
		a.mu.Unlock()
		return
	}
	a.last[kind] = a.now()
	a.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := a.send(ctx, "[Scraper] "+text); err != nil {
			slog.Error("Failed to send alert", "kind", kind, "error", err)
		}
	}()
}

// evolutionSender posts a WhatsApp text through Evolution API v2 (POST /message/sendText/{instance}).
func evolutionSender(baseURL, apiKey, instance, number string) func(ctx context.Context, text string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	return func(ctx context.Context, text string) error {
		body, _ := json.Marshal(map[string]string{"number": number, "text": text})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/message/sendText/"+instance, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("apikey", apiKey)
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			return fmt.Errorf("evolution api returned http %d", resp.StatusCode)
		}
		return nil
	}
}
