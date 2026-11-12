package service

import (
	"bytes"
	"encoding/csv"
	"testing"

	"business-scraping-service/backend/internal/model"
)

func TestBuildWAContactsCSV(t *testing.T) {
	hot := model.LeadStatusHot
	leads := []*model.BusinessLead{
		{Name: "Kopi Canggu", Category: "Cafe", City: "Canggu", PrimaryPhone: "0812-3456-7890", LeadStatus: &hot},
		{Name: "Landline Only", PrimaryPhone: "(0361) 123456"},
		{Name: "Mobile On Website", PrimaryPhone: "0361 123456", Phones: []string{"+62 857 1111 2222"}},
		{Name: "Duplicate Of Kopi", PrimaryPhone: "+6281234567890"},
		{Name: "No Phone"},
	}

	res, err := buildWAContactsCSV(leads, WAExportOptions{MobileOnly: true, StatusPrefix: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Exported != 2 || res.Skipped != 3 {
		t.Errorf("exported/skipped = %d/%d, want 2/3", res.Exported, res.Skipped)
	}

	rows, err := csv.NewReader(bytes.NewReader(res.CSV)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if rows[0][0] != "First Name" || rows[0][6] != "Phone 1 - Value" {
		t.Errorf("unexpected header: %v", rows[0])
	}
	if rows[1][0] != "[HOT] Kopi Canggu" || rows[1][6] != "+6281234567890" {
		t.Errorf("unexpected first row: %v", rows[1])
	}
	if rows[1][4] != "* myContacts ::: Momos Leads ::: Hot" {
		t.Errorf("unexpected labels: %q", rows[1][4])
	}
	if rows[2][0] != "Mobile On Website" || rows[2][6] != "+6285711112222" {
		t.Errorf("should fall back to the mobile found on the website: %v", rows[2])
	}

	res, _ = buildWAContactsCSV(leads, WAExportOptions{MobileOnly: false})
	if res.Exported != 3 {
		t.Errorf("without mobile_only the landline should be included, exported = %d", res.Exported)
	}
	rows, _ = csv.NewReader(bytes.NewReader(res.CSV)).ReadAll()
	if rows[2][0] != "Landline Only" || rows[2][5] != "Work" {
		t.Errorf("landline should be labelled Work: %v", rows[2])
	}
}
