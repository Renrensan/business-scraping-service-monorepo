package service

import (
	"errors"
	"strings"
	"testing"

	"business-scraping-service/backend/internal/apperr"
	"business-scraping-service/backend/internal/model"
)

func TestCleanContact(t *testing.T) {
	tests := []struct {
		name      string
		in        model.LeadContactInput
		wantPhone string
		wantEmail string
		wantErr   string
	}{
		{"full contact", model.LeadContactInput{Name: " Made Sudarsana ", Role: "GM", Phone: "0812-3456-7890", Email: " Made@Alaya.com "}, "+6281234567890", "made@alaya.com", ""},
		{"name only", model.LeadContactInput{Name: "Putu"}, "", "", ""},
		{"international phone kept", model.LeadContactInput{Name: "Anna", Phone: "+61 450 177 118"}, "+61450177118", "", ""},
		{"missing name", model.LeadContactInput{Phone: "0812"}, "", "", "name is required"},
		{"bad phone", model.LeadContactInput{Name: "X", Phone: "call me"}, "", "", "doesn't look like a phone number"},
		{"bad email", model.LeadContactInput{Name: "X", Email: "made@alaya"}, "", "", "doesn't look like an email"},
		{"email with display name", model.LeadContactInput{Name: "X", Email: "Made <made@alaya.com>"}, "", "", "doesn't look like an email"},
		{"note too long", model.LeadContactInput{Name: "X", Note: strings.Repeat("a", 2001)}, "", "", "note is too long"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cleanContact(tt.in)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) || !errors.Is(err, apperr.ErrInvalid) {
					t.Fatalf("err = %v, want invalid containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Phone != tt.wantPhone || got.Email != tt.wantEmail || got.Name != strings.TrimSpace(tt.in.Name) {
				t.Errorf("got %+v", got)
			}
		})
	}
}
