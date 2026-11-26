package scoring

import (
	"encoding/json"
	"os"
	"reflect"
	"regexp"
	"testing"
)

// The GuestChatPro profile seeded by migration 0005 must match GuestChatProConfig.
func TestSeededGuestChatProMatchesCode(t *testing.T) {
	raw, err := os.ReadFile("../repository/migrations/0005_lead_scoring.sql")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?s)'\["Lodging"\]',\s*'(\{.*?\})'\s*\)`).FindSubmatch(raw)
	if m == nil {
		t.Fatal("seed config not found")
	}
	var got Config
	if err := json.Unmarshal(m[1], &got); err != nil {
		t.Fatal(err)
	}
	if want := GuestChatProConfig(); !reflect.DeepEqual(got, want) {
		t.Errorf("seed differs from code:\n got %+v\nwant %+v", got, want)
	}
}
