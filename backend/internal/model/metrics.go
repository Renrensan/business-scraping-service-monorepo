package model

import "time"

type OverviewMetrics struct {
	TotalLeads          int64      `json:"total_leads"`
	EnrichedLeads       int64      `json:"enriched_leads"`
	LeadsWithEmail      int64      `json:"leads_with_email"`
	LeadsWithPhone      int64      `json:"leads_with_phone"`
	EmailHitRate        float64    `json:"email_hit_rate"`
	PhoneHitRate        float64    `json:"phone_hit_rate"`
	ActiveJobs          int64      `json:"active_jobs"`
	CompletedJobs       int64      `json:"completed_jobs"`
	TotalScrapedEver    int64      `json:"total_scraped_ever"`
	NextScrapeAt        *time.Time `json:"next_scrape_at,omitempty"`
	CountdownSeconds    int64      `json:"countdown_seconds"`
	AutoScheduleEnabled bool       `json:"auto_schedule_enabled"`
	ScheduleInterval    string     `json:"schedule_interval"`
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
