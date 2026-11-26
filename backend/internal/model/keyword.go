package model

import (
	"time"

	"github.com/google/uuid"
)

type KeywordStatus string

const (
	KeywordStatusReady      KeywordStatus = "ready"
	KeywordStatusProcessing KeywordStatus = "processing"
	KeywordStatusPaginating KeywordStatus = "paginating"
	KeywordStatusCompleted  KeywordStatus = "completed"
	// KeywordStatusNoResults: Google returned nothing on the first page. Shown so admins can fix the
	// term instead of the keyword silently looking "completed".
	KeywordStatusNoResults KeywordStatus = "no_results"
)

// NextKeywordStatus decides a keyword's status after a run that started at the first page or continued
// from a saved cursor.
func NextKeywordStatus(startedFromFirstPage bool, leadsFound int, nextToken string, cycleFinished bool) KeywordStatus {
	if startedFromFirstPage && leadsFound == 0 && (cycleFinished || nextToken == "") {
		return KeywordStatusNoResults
	}
	if cycleFinished || nextToken == "" {
		return KeywordStatusCompleted
	}
	return KeywordStatusPaginating
}

type ScraperKeyword struct {
	ID                uuid.UUID     `json:"id"`
	Keyword           string        `json:"keyword"`
	NextPageToken     string        `json:"next_page_token"`
	CurrentPage       int           `json:"current_page"`
	TotalLeadsScraped int           `json:"total_leads_scraped"`
	LastScrapedAt     *time.Time    `json:"last_scraped_at,omitempty"`
	Status            KeywordStatus `json:"status"`
	IsActive          bool          `json:"is_active"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

type CreateKeywordRequest struct {
	Keywords []string `json:"keywords"`
	Keyword  string   `json:"keyword,omitempty"`
}

type ListKeywordFilter struct {
	Search   string
	Status   string
	Page     int
	PageSize int
}

type ListKeywordResponse struct {
	Items      []*ScraperKeyword `json:"items"`
	TotalCount int64             `json:"total_count"`
	TotalPages int               `json:"total_pages"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
}
