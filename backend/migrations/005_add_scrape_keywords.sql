-- Scrape keywords live in their own column, separate from Google's categories.
-- Previously every scrape appended its keyword (e.g. "petshop in canggu") to subcategories, so the
-- Category filter matched any business Google returned for that search (hotels under "Petshop").
ALTER TABLE business_leads ADD COLUMN IF NOT EXISTS scrape_keywords JSONB DEFAULT '[]'::jsonb;
UPDATE business_leads SET scrape_keywords = '[]'::jsonb WHERE scrape_keywords IS NULL OR jsonb_typeof(scrape_keywords) != 'array';
CREATE INDEX IF NOT EXISTS idx_business_leads_scrape_keywords ON business_leads USING GIN (scrape_keywords);

-- Move keyword tags out of subcategories into scrape_keywords (idempotent: no-op once moved).
-- A tag equal to the lead's own category is left alone: it is also a real Google type.
WITH tags AS (
    SELECT keyword AS tag FROM scraper_keywords WHERE keyword != ''
    UNION SELECT query FROM scrape_jobs WHERE query != ''
    UNION SELECT category FROM scrape_jobs WHERE category != ''
)
UPDATE business_leads bl SET
    scrape_keywords = COALESCE((
        SELECT jsonb_agg(DISTINCT x.v) FROM (
            SELECT k AS v FROM jsonb_array_elements_text(bl.scrape_keywords) AS k
            UNION
            SELECT s FROM jsonb_array_elements_text(bl.subcategories) AS s
            WHERE s IN (SELECT tag FROM tags) AND s != 'New Business' AND LOWER(s) != LOWER(bl.category)
        ) x
    ), '[]'::jsonb),
    subcategories = COALESCE((
        SELECT jsonb_agg(t.s ORDER BY t.ord)
        FROM jsonb_array_elements_text(bl.subcategories) WITH ORDINALITY AS t(s, ord)
        WHERE t.s NOT IN (SELECT tag FROM tags) OR t.s = 'New Business' OR LOWER(t.s) = LOWER(bl.category)
    ), '[]'::jsonb)
WHERE EXISTS (
    SELECT 1 FROM jsonb_array_elements_text(bl.subcategories) AS s
    WHERE s IN (SELECT tag FROM tags) AND s != 'New Business' AND LOWER(s) != LOWER(bl.category)
);

-- Leads whose category was overwritten with a raw search query (lower-case, e.g. "petshop in canggu")
-- get their first real Google type back.
UPDATE business_leads bl SET category = COALESCE((
        SELECT t.s FROM jsonb_array_elements_text(bl.subcategories) WITH ORDINALITY AS t(s, ord)
        WHERE t.s != 'New Business' ORDER BY t.ord LIMIT 1
    ), 'Local Business')
WHERE bl.category ~ '^[a-z]'
  AND bl.category IN (SELECT query FROM scrape_jobs WHERE query != '' UNION SELECT keyword FROM scraper_keywords WHERE keyword != '');
