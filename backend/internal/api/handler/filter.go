package handler

import (
	"net/url"
	"strconv"
	"strings"

	"business-scraping-service/backend/internal/model"
	"github.com/google/uuid"
)

// parseBusinessFilter reads the lead filters shared by the list and export endpoints,
// so both always select exactly the same leads.
//
// Multi-value params accept repeated keys (keywords=a&keywords=b) or a comma list (status=cold,hot).
// Keywords are not comma-split because a scrape keyword may itself contain a comma.
func parseBusinessFilter(q url.Values) model.BusinessFilter {
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))

	filter := model.BusinessFilter{
		Search:    q.Get("search"),
		Category:  q.Get("category"),
		City:      q.Get("city"),
		State:     q.Get("state"),
		SortBy:    q.Get("sort_by"),
		SortOrder: q.Get("sort_order"),
		Page:      page,
		PageSize:  pageSize,
	}

	// "keywords" (multi) plus the legacy single "keyword" param.
	filter.Keywords = append(filter.Keywords, q["keywords"]...)
	filter.Keywords = append(filter.Keywords, q["keywords[]"]...)
	if kw := q.Get("keyword"); kw != "" {
		filter.Keywords = append(filter.Keywords, kw)
	}

	for _, raw := range append(q["status"], q["status[]"]...) {
		for _, part := range strings.Split(raw, ",") {
			st := model.LeadStatus(strings.ToLower(strings.TrimSpace(part)))
			if st.Valid() {
				filter.Statuses = append(filter.Statuses, st)
			}
		}
	}

	filter.HasEmail = parseBoolParam(q, "has_email")
	filter.HasPhone = parseBoolParam(q, "has_phone")
	filter.HasWebsite = parseBoolParam(q, "has_website")
	filter.Contacted = parseBoolParam(q, "contacted")
	filter.OnlyNew = parseBoolParam(q, "only_new")

	if v := q.Get("min_rating"); v != "" {
		if val, err := strconv.ParseFloat(v, 64); err == nil {
			filter.MinRating = &val
		}
	}
	if v := q.Get("max_reviews"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			filter.MaxReviews = &val
		}
	}
	// Scoring: score_profile is a profile ID or name; tiers can repeat or be comma-separated.
	if v := strings.TrimSpace(q.Get("score_profile")); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.ScoreProfileID = &id
		} else {
			filter.ScoreProfileName = v
		}
	}
	for _, raw := range append(q["tiers"], q["tiers[]"]...) {
		for _, part := range strings.Split(raw, ",") {
			if t := strings.ToLower(strings.TrimSpace(part)); t != "" {
				filter.Tiers = append(filter.Tiers, t)
			}
		}
	}
	filter.ScoreArea = q.Get("score_area")

	if v := q.Get("job_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.JobID = &id
		}
	}

	return filter
}

func parseBoolParam(q url.Values, key string) *bool {
	v := q.Get(key)
	if v == "" {
		return nil
	}
	b := v == "true"
	return &b
}
