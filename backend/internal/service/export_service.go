package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"

	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"
	"business-scraping-service/backend/internal/utils"
)

type ExportService struct {
	businessRepo *repository.BusinessRepository
}

func NewExportService(businessRepo *repository.BusinessRepository) *ExportService {
	return &ExportService{businessRepo: businessRepo}
}

func (s *ExportService) ExportCSV(ctx context.Context, filter model.BusinessFilter) ([]byte, string, error) {
	leads, err := s.businessRepo.GetAllForExport(ctx, filter)
	if err != nil {
		return nil, "", err
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	headers := []string{
		"Name", "Category", "Address", "City", "State", "Postal Code", "Country",
		"Phone", "Website", "Emails", "Secondary Phones", "Rating", "Review Count",
		"Price Level", "Claimed", "Contacted", "Lead Status", "Facebook", "Instagram", "LinkedIn", "Twitter",
		"YouTube", "TikTok", "Google Maps URL", "Latitude", "Longitude", "Place ID",
	}
	if err := writer.Write(headers); err != nil {
		return nil, "", err
	}

	for _, l := range leads {
		emailsStr := strings.Join(l.Emails, "; ")
		phonesStr := strings.Join(l.Phones, "; ")
		row := []string{
			l.Name,
			l.Category,
			l.Address,
			l.City,
			l.State,
			l.PostalCode,
			l.Country,
			l.PrimaryPhone,
			l.Website,
			emailsStr,
			phonesStr,
			fmt.Sprintf("%.1f", l.Rating),
			fmt.Sprintf("%d", l.ReviewCount),
			l.PriceLevel,
			fmt.Sprintf("%t", l.Claimed),
			fmt.Sprintf("%t", l.Contacted),
			leadStatusText(l.LeadStatus),
			l.FacebookURL,
			l.InstagramURL,
			l.LinkedinURL,
			l.TwitterURL,
			l.YoutubeURL,
			l.TiktokURL,
			l.GoogleMapsURL,
			fmt.Sprintf("%f", l.Latitude),
			fmt.Sprintf("%f", l.Longitude),
			l.PlaceID,
		}
		if err := writer.Write(row); err != nil {
			return nil, "", err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", err
	}

	filename := "business_leads_export.csv"
	return buf.Bytes(), filename, nil
}

func leadStatusText(s *model.LeadStatus) string {
	if s == nil {
		return ""
	}
	return string(*s)
}

// WAExportOptions controls the WhatsApp contacts export.
type WAExportOptions struct {
	MobileOnly   bool // skip landlines / toll-free numbers (they can't have WhatsApp)
	StatusPrefix bool // prefix names with [HOT]/[WARM]/[COLD] so status shows in the WA chat list
}

// WAExportResult is a Google Contacts CSV plus how many leads made it in.
type WAExportResult struct {
	CSV      []byte
	Exported int
	Skipped  int // leads without a usable (mobile) number, or whose number was already exported
}

// waContactsLabel is the Google Contacts label (group) every exported lead gets.
const waContactsLabel = "Momos Leads"

// ExportWAContacts builds a CSV in Google Contacts' import format. Imported into Google
// Contacts, the numbers sync to the phone and show up in WhatsApp / WhatsApp Business.
func (s *ExportService) ExportWAContacts(ctx context.Context, filter model.BusinessFilter, opts WAExportOptions) (*WAExportResult, error) {
	leads, err := s.businessRepo.GetAllForExport(ctx, filter)
	if err != nil {
		return nil, err
	}
	return buildWAContactsCSV(leads, opts)
}

func buildWAContactsCSV(leads []*model.BusinessLead, opts WAExportOptions) (*WAExportResult, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	headers := []string{
		"First Name", "Last Name", "Organization Name", "Notes", "Labels",
		"Phone 1 - Label", "Phone 1 - Value",
	}
	if err := writer.Write(headers); err != nil {
		return nil, err
	}

	res := &WAExportResult{}
	seen := make(map[string]bool)

	for _, l := range leads {
		number, isMobile := pickWANumber(l, opts.MobileOnly)
		if number == "" || seen[number] {
			res.Skipped++
			continue
		}
		seen[number] = true

		name := strings.TrimSpace(l.Name)
		if opts.StatusPrefix && l.LeadStatus != nil {
			name = "[" + strings.ToUpper(string(*l.LeadStatus)) + "] " + name
		}

		labels := []string{"* myContacts", waContactsLabel}
		if l.LeadStatus != nil {
			labels = append(labels, strings.ToUpper(string((*l.LeadStatus)[:1]))+string((*l.LeadStatus)[1:]))
		}

		var notes []string
		if l.Category != "" {
			notes = append(notes, l.Category)
		}
		if l.City != "" {
			notes = append(notes, l.City)
		}
		if l.Rating > 0 {
			notes = append(notes, fmt.Sprintf("Rating %.1f (%d reviews)", l.Rating, l.ReviewCount))
		}
		if l.Website != "" {
			notes = append(notes, l.Website)
		}
		if l.GoogleMapsURL != "" {
			notes = append(notes, l.GoogleMapsURL)
		}

		row := []string{
			name,
			"",
			l.Category,
			strings.Join(notes, " | "),
			strings.Join(labels, " ::: "),
			phoneLabel(isMobile),
			number,
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
		res.Exported++
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	res.CSV = buf.Bytes()
	return res, nil
}

// pickWANumber returns the lead's best WhatsApp number: the first mobile among the primary
// phone and the extra phones found on its website. Without mobileOnly, a landline is used
// when no mobile exists.
func pickWANumber(l *model.BusinessLead, mobileOnly bool) (number string, isMobile bool) {
	candidates := append([]string{l.PrimaryPhone}, l.Phones...)
	fallback := ""
	for _, c := range candidates {
		number, isMobile, ok := utils.ToWhatsAppNumber(c)
		if !ok {
			continue
		}
		if isMobile {
			return number, true
		}
		if fallback == "" {
			fallback = number
		}
	}
	if mobileOnly {
		return "", false
	}
	return fallback, false
}

func phoneLabel(isMobile bool) string {
	if isMobile {
		return "Mobile"
	}
	return "Work"
}
