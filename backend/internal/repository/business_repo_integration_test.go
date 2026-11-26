package repository

import (
	"context"
	"testing"

	"business-scraping-service/backend/internal/model"
)

// A stored list in the wrong JSON shape must stop the merge: merging as if it were empty
// would overwrite the lead's real emails on the next scrape.
func TestUpsert_UnreadableStoredListIsAnErrorNotDataLoss(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	leads := NewBusinessRepository(pool)

	lead := &model.BusinessLead{PlaceID: "places/x1", Name: "Villa X", Address: "Jl. X", Emails: []string{"a@villa.com"}}
	if _, err := leads.Upsert(ctx, lead); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE business_leads SET emails = '"a@villa.com"'::jsonb WHERE id = $1`, lead.ID); err != nil {
		t.Fatal(err)
	}

	again := &model.BusinessLead{PlaceID: "places/x1", Name: "Villa X", Address: "Jl. X", Emails: []string{"b@villa.com"}}
	if _, err := leads.Upsert(ctx, again); err == nil {
		t.Fatal("expected an error for an unreadable stored emails list")
	}
	var stored string
	_ = pool.QueryRow(ctx, `SELECT emails::text FROM business_leads WHERE id = $1`, lead.ID).Scan(&stored)
	if stored != `"a@villa.com"` {
		t.Errorf("stored emails changed to %s", stored)
	}
}

// Reading a lead with one unreadable list still returns the lead (the field is left empty).
func TestGetByID_UnreadableListDoesNotFailTheRead(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	leads := NewBusinessRepository(pool)

	lead := &model.BusinessLead{PlaceID: "places/x2", Name: "Villa Y", Address: "Jl. Y", Emails: []string{"a@y.com"}}
	if _, err := leads.Upsert(ctx, lead); err != nil {
		t.Fatal(err)
	}
	_, _ = pool.Exec(ctx, `UPDATE business_leads SET photos = '{"not":"a list"}'::jsonb WHERE id = $1`, lead.ID)

	got, err := leads.GetByID(ctx, lead.ID)
	if err != nil || got == nil || len(got.Emails) != 1 || got.Photos != nil {
		t.Fatalf("got %+v, err %v", got, err)
	}
	list, err := leads.List(ctx, model.BusinessFilter{Page: 1, PageSize: 10})
	if err != nil || list.TotalCount != 1 {
		t.Fatalf("list: %v %v", list, err)
	}
}
