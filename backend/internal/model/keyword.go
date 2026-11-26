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

// KeywordMode: "plain" keywords send their text as-is (up to 60 results per cycle); "area" keywords
// search a term inside an area's rectangle, splitting busy areas into smaller cells to get past 60.
type KeywordMode string

const (
	KeywordModePlain KeywordMode = "plain"
	KeywordModeArea  KeywordMode = "area"
)

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

	Mode             KeywordMode `json:"mode"`
	Term             string      `json:"term,omitempty"`
	IncludedType     string      `json:"included_type,omitempty"`
	AreaID           *uuid.UUID  `json:"area_id,omitempty"`
	AreaName         string      `json:"area_name,omitempty"`
	CellsTotal       int         `json:"cells_total"`
	CellsDone        int         `json:"cells_done"`
	NewPlacesLastRun int         `json:"new_places_last_run"`
	EmptyRuns        int         `json:"empty_runs"`
	NextEligibleAt   *time.Time  `json:"next_eligible_at,omitempty"`
}

type CreateKeywordRequest struct {
	Keywords []string `json:"keywords"`
	Keyword  string   `json:"keyword,omitempty"`

	// Area mode: one keyword per area.
	Term         string      `json:"term,omitempty"`
	IncludedType string      `json:"included_type,omitempty"`
	AreaIDs      []uuid.UUID `json:"area_ids,omitempty"`
}

// Area is a named map rectangle that area keywords search inside.
type Area struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	LowLat    float64   `json:"low_lat"`
	LowLng    float64   `json:"low_lng"`
	HighLat   float64   `json:"high_lat"`
	HighLng   float64   `json:"high_lng"`
	Source    string    `json:"source"` // preset | lookup | manual
	Keywords  int       `json:"keywords"`
	CreatedAt time.Time `json:"created_at"`
}

// KeywordCell is one rectangle of an area keyword's current cycle.
type KeywordCell struct {
	ID            uuid.UUID `json:"id"`
	KeywordID     uuid.UUID `json:"keyword_id"`
	Depth         int       `json:"depth"`
	LowLat        float64   `json:"low_lat"`
	LowLng        float64   `json:"low_lng"`
	HighLat       float64   `json:"high_lat"`
	HighLng       float64   `json:"high_lng"`
	Status        string    `json:"status"` // pending | done | split
	NextPageToken string    `json:"next_page_token"`
	PagesFetched  int       `json:"pages_fetched"`
	Results       int       `json:"results"`
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
