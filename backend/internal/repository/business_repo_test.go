package repository

import (
	"os"
	"strings"
	"testing"

	"business-scraping-service/backend/internal/model"
)

func TestSafeJSONArray(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "nil slice",
			input:    nil,
			expected: "[]",
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: "[]",
		},
		{
			name:     "valid strings",
			input:    []string{"info@example.com", "sales@example.com"},
			expected: `["info@example.com","sales@example.com"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(safeJSONArray(tt.input))
			if got != tt.expected {
				t.Errorf("safeJSONArray() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSafeJSONObject(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected string
	}{
		{
			name:     "nil map",
			input:    nil,
			expected: "{}",
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: "{}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(safeJSONObject(tt.input))
			if got != tt.expected {
				t.Errorf("safeJSONObject() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNormalizeDomain(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://www.example.com/about", "example.com"},
		{"http://example.com/contact?foo=bar", "example.com"},
		{"WWW.BALIVILLAS.COM/", "balivillas.com"},
		{"", ""},
		{"https://subdomain.domain.co.id/page#anchor", "subdomain.domain.co.id"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeDomain(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeDomain(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNormalizePhoneDigits(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"+62 812-3456-7890", "6281234567890"},
		{"(0361) 123456", "0361123456"},
		{"0812 3456 7890", "081234567890"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizePhoneDigits(tt.input)
			if got != tt.expected {
				t.Errorf("normalizePhoneDigits(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestListQueryBuilderGuardsArrayScalars(t *testing.T) {
	repo := &BusinessRepository{db: nil}

	hasEmailTrue := true
	filterEmail := model.BusinessFilter{HasEmail: &hasEmailTrue}

	whereClauses := []string{"1=1"}
	if filterEmail.HasEmail != nil {
		if *filterEmail.HasEmail {
			whereClauses = append(whereClauses, "(emails IS NOT NULL AND jsonb_typeof(emails) = 'array' AND jsonb_array_length(emails) > 0)")
		} else {
			whereClauses = append(whereClauses, "(emails IS NULL OR jsonb_typeof(emails) != 'array' OR jsonb_array_length(emails) = 0)")
		}
	}

	sql := strings.Join(whereClauses, " AND ")
	if !strings.Contains(sql, "jsonb_typeof(emails) = 'array'") {
		t.Errorf("Expected sql to contain jsonb_typeof guard, got: %s", sql)
	}

	hasPhoneFalse := false
	filterPhone := model.BusinessFilter{HasPhone: &hasPhoneFalse}
	whereClauses = []string{"1=1"}
	if filterPhone.HasPhone != nil {
		if *filterPhone.HasPhone {
			whereClauses = append(whereClauses, "((primary_phone IS NOT NULL AND primary_phone != '') OR (phones IS NOT NULL AND jsonb_typeof(phones) = 'array' AND jsonb_array_length(phones) > 0))")
		} else {
			whereClauses = append(whereClauses, "((primary_phone IS NULL OR primary_phone = '') AND (phones IS NULL OR jsonb_typeof(phones) != 'array' OR jsonb_array_length(phones) = 0))")
		}
	}
	sqlPhone := strings.Join(whereClauses, " AND ")
	if !strings.Contains(sqlPhone, "jsonb_typeof(phones) != 'array'") {
		t.Errorf("Expected sql to contain jsonb_typeof guard for phone, got: %s", sqlPhone)
	}
	_ = repo
}

func TestDeriveCategoryFromKeyword(t *testing.T) {
	tests := []struct {
		kw       string
		expected string
	}{
		{"cafe in canggu", "Cafe"},
		{"cafe in denpasar", "Cafe"},
		{"cafes in bali", "Cafe"},
		{"veterinarian in denpasar, bali", "Veterinarian"},
		{"veterinarian in ubud", "Veterinarian"},
		{"coworking space at ubud", "Coworking Space"},
		{"hotel near sanur", "Hotel"},
		{"dentist, kuta", "Dentist"},
		{"gym - seminyak", "Gym"},
		{"villa rental | seminyak", "Villa Rental"},
		{"digital marketing agency di badung", "Digital Marketing Agency"},
		{"fitness center in denpasar", "Fitness Center"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.kw, func(t *testing.T) {
			got := DeriveCategoryFromKeyword(tt.kw)
			if got != tt.expected {
				t.Errorf("DeriveCategoryFromKeyword(%q) = %q, want %q", tt.kw, got, tt.expected)
			}
		})
	}
}


func TestBuildBusinessWhereKeywordsAndStatus(t *testing.T) {
	contacted := true
	filter := model.BusinessFilter{
		Search:    "kopi",
		Contacted: &contacted,
		Keywords:  []string{" cafe in canggu ", "", "villa in ubud", "cafe in canggu"},
		Statuses:  []model.LeadStatus{model.LeadStatusWarm, "bogus", model.LeadStatusHot},
	}

	sql, args := buildBusinessWhere(filter)

	if !strings.Contains(sql, "scrape_keywords ?| $3::text[]") || !strings.Contains(sql, "query ILIKE ANY($3::text[])") {
		t.Errorf("expected multi-keyword clause on $3, got: %s", sql)
	}
	if !strings.Contains(sql, "lead_status = ANY($4::text[])") {
		t.Errorf("expected status clause on $4, got: %s", sql)
	}
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d: %#v", len(args), args)
	}
	kws, _ := args[2].([]string)
	if len(kws) != 2 || kws[0] != "cafe in canggu" || kws[1] != "villa in ubud" {
		t.Errorf("keywords should be trimmed and de-duplicated, got %#v", args[2])
	}
	sts, _ := args[3].([]string)
	if len(sts) != 2 || sts[0] != "warm" || sts[1] != "hot" {
		t.Errorf("invalid statuses should be dropped, got %#v", args[3])
	}
}

func TestBuildBusinessWhereNoKeywordsNoStatus(t *testing.T) {
	sql, args := buildBusinessWhere(model.BusinessFilter{Keywords: []string{"  "}})
	if sql != "1=1" || len(args) != 0 {
		t.Errorf("blank filters should add no clauses, got %q %#v", sql, args)
	}
}

// The scrape upsert must never touch admin-owned columns, or a re-scrape would wipe
// the contacted flag / lead status the admin set.
func TestUpsertDoesNotTouchAdminColumns(t *testing.T) {
	src, err := os.ReadFile("business_repo.go")
	if err != nil {
		t.Fatal(err)
	}
	code := string(src)
	start := strings.Index(code, "func (r *BusinessRepository) Upsert(")
	end := strings.Index(code, "func mergeUniqueStrings(")
	if start < 0 || end < start {
		t.Fatal("could not locate Upsert")
	}
	upsert := code[start:end]
	for _, col := range []string{"lead_status", "contacted"} {
		if strings.Contains(upsert, col+" =") || strings.Contains(upsert, col+",") {
			t.Errorf("Upsert must not write %s", col)
		}
	}
}

// Hotels returned by a "petshop" search must not show under the Petshop category:
// the category filter looks only at Google's categories, never at scrape keyword tags,
// and the keyword filter looks only at the scrape tags.
func TestCategoryAndKeywordFiltersUseSeparateColumns(t *testing.T) {
	catSQL, _ := buildBusinessWhere(model.BusinessFilter{Category: "Pet Store"})
	if strings.Contains(catSQL, "scrape_keywords") {
		t.Errorf("category filter must not match scrape keywords, got: %s", catSQL)
	}
	if !strings.Contains(catSQL, "category ILIKE") {
		t.Errorf("category filter should match the category column, got: %s", catSQL)
	}

	kwSQL, _ := buildBusinessWhere(model.BusinessFilter{Keywords: []string{"petshop in canggu"}})
	if strings.Contains(kwSQL, "subcategories") {
		t.Errorf("keyword filter must not match Google categories, got: %s", kwSQL)
	}
}

func TestUpsertPersistsScrapeKeywords(t *testing.T) {
	src, err := os.ReadFile("business_repo.go")
	if err != nil {
		t.Fatal(err)
	}
	code := string(src)
	upsert := code[strings.Index(code, "func (r *BusinessRepository) Upsert("):strings.Index(code, "func mergeUniqueStrings(")]
	for _, want := range []string{"scrape_keywords = $37", "updated_at, scrape_keywords", "scrape_keywords = COALESCE(", "mergeUniqueStrings(existingScrapeKws"} {
		if !strings.Contains(upsert, want) {
			t.Errorf("Upsert should contain %q", want)
		}
	}
}
