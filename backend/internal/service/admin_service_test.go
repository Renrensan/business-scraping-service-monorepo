package service

import "testing"

func TestNormalizeAdmin(t *testing.T) {
	tests := []struct {
		email, username, wantUser string
		ok                        bool
	}{
		{"adiztraa@gmail.com", "", "adiztraa", true},
		{"Brewpending@Gmail.com", "", "brewpending", true},
		{"first.last+tag@gmail.com", "", "first.last-tag", true},
		{"z@example.com", "", "z", true},
		{"ops@example.com", "Ops Team", "", false},
		{"not-an-email", "", "", false},
	}
	for _, tt := range tests {
		got, err := normalizeAdmin(CreateAdminInput{Email: tt.email, Username: tt.username, Password: "test-password-123"})
		if (err == nil) != tt.ok || (tt.ok && got.Username != tt.wantUser) {
			t.Errorf("%s/%q: got %q err %v", tt.email, tt.username, got.Username, err)
		}
	}
}
