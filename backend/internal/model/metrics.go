package model

type OverviewMetrics struct {
	TotalLeads       int64   `json:"total_leads"`
	EnrichedLeads    int64   `json:"enriched_leads"`
	EmailHitRate     float64 `json:"email_hit_rate"`
	PhoneHitRate     float64 `json:"phone_hit_rate"`
	ActiveJobs       int64   `json:"active_jobs"`
	CompletedJobs    int64   `json:"completed_jobs"`
	TotalScrapedEver int64   `json:"total_scraped_ever"`
}

type CategoryStat struct {
	Category string `json:"category"`
	Count    int64  `json:"count"`
}

type LocationStat struct {
	City  string `json:"city"`
	State string `json:"state"`
	Count int64  `json:"count"`
}
