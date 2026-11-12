package handler

import (
	"net/url"
	"reflect"
	"testing"

	"business-scraping-service/backend/internal/model"
)

func TestParseBusinessFilterMultiValues(t *testing.T) {
	q, _ := url.ParseQuery("keywords=cafe+in+canggu&keywords=bar%2C+pub+in+kuta&keyword=villa&status=cold,HOT&status=nope&contacted=true&min_rating=4.5")
	f := parseBusinessFilter(q)

	wantKw := []string{"cafe in canggu", "bar, pub in kuta", "villa"}
	if !reflect.DeepEqual(f.Keywords, wantKw) {
		t.Errorf("Keywords = %#v, want %#v", f.Keywords, wantKw)
	}
	wantSt := []model.LeadStatus{model.LeadStatusCold, model.LeadStatusHot}
	if !reflect.DeepEqual(f.Statuses, wantSt) {
		t.Errorf("Statuses = %#v, want %#v", f.Statuses, wantSt)
	}
	if f.Contacted == nil || !*f.Contacted {
		t.Error("contacted=true not parsed")
	}
	if f.MinRating == nil || *f.MinRating != 4.5 {
		t.Error("min_rating not parsed")
	}
	if f.HasEmail != nil {
		t.Error("absent has_email should stay nil")
	}
}
