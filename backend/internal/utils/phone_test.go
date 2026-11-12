package utils

import "testing"

func TestToWhatsAppNumber(t *testing.T) {
	tests := []struct {
		in         string
		wantNumber string
		wantMobile bool
		wantOK     bool
	}{
		{"0812-3456-7890", "+6281234567890", true, true},
		{"0812 3456 789", "+628123456789", true, true},
		{"+62 812-3456-7890", "+6281234567890", true, true},
		{"6281234567890", "+6281234567890", true, true},
		{"81234567890", "+6281234567890", true, true},
		{"(0361) 123456", "+62361123456", false, true}, // Bali landline
		{"021 5551234", "+62215551234", false, true},   // Jakarta landline
		{"0800 1234567", "+628001234567", false, true}, // toll-free
		{"0061 2 9876 5432", "+61298765432", true, true},
		{"+44 20 7946 0958", "+442079460958", true, true},
		{"", "", false, false},
		{"12345", "", false, false},
		{"0812", "", false, false},
		{"not a phone", "", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			n, mobile, ok := ToWhatsAppNumber(tt.in)
			if n != tt.wantNumber || mobile != tt.wantMobile || ok != tt.wantOK {
				t.Errorf("ToWhatsAppNumber(%q) = (%q, %v, %v), want (%q, %v, %v)",
					tt.in, n, mobile, ok, tt.wantNumber, tt.wantMobile, tt.wantOK)
			}
		})
	}
}
