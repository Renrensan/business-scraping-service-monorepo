package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"business-scraping-service/backend/internal/model"
)

func TestContacts_CRUDListSummaryAndSearch(t *testing.T) {
	pool := testDB(t)
	ctx := context.Background()
	leads := NewBusinessRepository(pool)
	contacts := NewContactRepository(pool)

	alaya := &model.BusinessLead{PlaceID: "p-alaya", Name: "Alaya Resort Ubud", Address: "Jl. Hanoman"}
	other := &model.BusinessLead{PlaceID: "p-other", Name: "Other Hotel", Address: "Jl. Lain"}
	for _, l := range []*model.BusinessLead{alaya, other} {
		if _, err := leads.Upsert(ctx, l); err != nil {
			t.Fatal(err)
		}
	}

	gm, err := contacts.Create(ctx, alaya.ID, model.LeadContactInput{Name: "Made Sudarsana", Role: "GM", Phone: "+6281234567890"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := contacts.Create(ctx, alaya.ID, model.LeadContactInput{Name: "Putu", Role: "Reservations"}); err != nil {
		t.Fatal(err)
	}
	if _, err := contacts.Create(ctx, uuid.New(), model.LeadContactInput{Name: "Ghost"}); !errors.Is(err, ErrLeadNotFound) {
		t.Errorf("contact for missing lead: %v", err)
	}

	list, _ := contacts.List(ctx, alaya.ID)
	if len(list) != 2 || list[0].Name != "Made Sudarsana" {
		t.Fatalf("list %+v", list)
	}

	// List rows carry the main (first) contact and the count.
	page, err := leads.List(ctx, model.BusinessFilter{Page: 1, PageSize: 10, SortBy: "name", SortOrder: "ASC"})
	if err != nil {
		t.Fatal(err)
	}
	row := page.Items[0]
	if row.ID != alaya.ID || row.ContactCount != 2 || row.MainContactName != "Made Sudarsana" || row.MainContactRole != "GM" {
		t.Errorf("summary: %+v", row)
	}
	if page.Items[1].ContactCount != 0 || page.Items[1].MainContactName != "" {
		t.Errorf("lead without contacts: %+v", page.Items[1])
	}

	// Search finds leads by contact name, phone or role.
	for _, q := range []string{"sudarsana", "81234567", "reservations"} {
		res, err := leads.List(ctx, model.BusinessFilter{Search: q, Page: 1, PageSize: 10})
		if err != nil || res.TotalCount != 1 || res.Items[0].ID != alaya.ID {
			t.Errorf("search %q: %+v %v", q, res, err)
		}
	}

	// Update and delete are scoped to the lead.
	if _, err := contacts.Update(ctx, other.ID, gm.ID, model.LeadContactInput{Name: "Hijack"}); !errors.Is(err, ErrContactNotFound) {
		t.Errorf("update via wrong lead: %v", err)
	}
	upd, err := contacts.Update(ctx, alaya.ID, gm.ID, model.LeadContactInput{Name: "Made S.", Role: "Owner", Note: "Decides for both Alaya properties"})
	if err != nil || upd.Role != "Owner" || upd.Note == "" || !upd.UpdatedAt.After(upd.CreatedAt) {
		t.Errorf("update: %+v %v", upd, err)
	}
	if err := contacts.Delete(ctx, other.ID, gm.ID); !errors.Is(err, ErrContactNotFound) {
		t.Errorf("delete via wrong lead: %v", err)
	}
	if err := contacts.Delete(ctx, alaya.ID, gm.ID); err != nil {
		t.Fatal(err)
	}

	// Deleting a lead removes its contacts.
	if err := leads.Delete(ctx, alaya.ID); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM lead_contacts`).Scan(&n)
	if n != 0 {
		t.Errorf("%d contacts left after deleting the lead", n)
	}
}
