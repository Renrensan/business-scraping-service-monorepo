package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"

	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/repository"
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
		"Price Level", "Claimed", "Contacted", "Facebook", "Instagram", "LinkedIn", "Twitter",
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
