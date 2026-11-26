package engine

import (
	"context"
	"testing"
	"time"
)

func TestNextQuotaReset_IsNextMidnightPacific(t *testing.T) {
	cases := []struct{ now, want string }{
		// 10:15 UTC = 02:15 PST -> next midnight PST = 08:00 UTC next day
		{"2026-11-26T10:15:00Z", "2026-11-27T08:00:00Z"},
		// 07:59 UTC = 23:59 PST previous day -> 08:00 UTC same day
		{"2026-11-26T07:59:00Z", "2026-11-26T08:00:00Z"},
		// summer (PDT, UTC-7): 12:00 UTC = 05:00 PDT -> 07:00 UTC next day
		{"2026-07-01T12:00:00Z", "2026-07-02T07:00:00Z"},
	}
	for _, c := range cases {
		now, _ := time.Parse(time.RFC3339, c.now)
		want, _ := time.Parse(time.RFC3339, c.want)
		if got := NextQuotaReset(now); !got.Equal(want) {
			t.Errorf("NextQuotaReset(%s) = %s, want %s", c.now, got, want)
		}
	}
}

type memQuotaStore struct{ until *time.Time }

func (m *memQuotaStore) SetQuotaPausedUntil(ctx context.Context, until *time.Time) error {
	m.until = until
	return nil
}

func TestQuotaGuard(t *testing.T) {
	store := &memQuotaStore{}
	g := NewQuotaGuard(store, nil)
	now := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)

	if _, paused := g.PausedUntil(now); paused {
		t.Fatal("new guard should not be paused")
	}
	until := g.Pause(context.Background(), now)
	if got, paused := g.PausedUntil(now); !paused || !got.Equal(until) || store.until == nil {
		t.Fatalf("pause not active/persisted: %v %v %v", got, paused, store.until)
	}
	if _, paused := g.PausedUntil(until.Add(time.Second)); paused {
		t.Error("pause should end at the reset time")
	}
	g.Clear(context.Background())
	if _, paused := g.PausedUntil(now); paused || store.until != nil {
		t.Error("clear should end the pause and persist nil")
	}

	// A pause loaded from the database (restart) is honoured.
	restored := NewQuotaGuard(store, &until)
	if _, paused := restored.PausedUntil(now); !paused {
		t.Error("restored pause not active")
	}
}

func TestMetaInt(t *testing.T) {
	m := map[string]any{"a": float64(3), "b": 4, "c": "x"}
	if metaInt(m, "a", 1) != 3 || metaInt(m, "b", 1) != 4 || metaInt(m, "c", 1) != 1 || metaInt(nil, "z", 7) != 7 {
		t.Error("metaInt")
	}
}
