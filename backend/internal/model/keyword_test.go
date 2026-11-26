package model

import "testing"

func TestNextKeywordStatus(t *testing.T) {
	cases := []struct {
		name      string
		fromFirst bool
		leads     int
		next      string
		finished  bool
		want      KeywordStatus
	}{
		{"empty first page", true, 0, "", true, KeywordStatusNoResults},
		{"full cycle", true, 57, "", true, KeywordStatusCompleted},
		{"more pages", true, 20, "tok", false, KeywordStatusPaginating},
		{"continued run ends empty", false, 0, "", true, KeywordStatusCompleted},
	}
	for _, c := range cases {
		if got := NextKeywordStatus(c.fromFirst, c.leads, c.next, c.finished); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
