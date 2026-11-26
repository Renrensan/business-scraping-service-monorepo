-- 1. Admin Users Table (Single User / Super Admin Auth)
CREATE TABLE IF NOT EXISTS admin_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(100) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Scrape Jobs Table
CREATE TABLE IF NOT EXISTS scrape_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    query VARCHAR(255) NOT NULL,
    location VARCHAR(255) NOT NULL,
    category VARCHAR(255) NOT NULL DEFAULT '',
    max_results INT NOT NULL DEFAULT 100,
    enrich_contacts BOOLEAN NOT NULL DEFAULT true,
    total_scraped INT NOT NULL DEFAULT 0,
    total_enriched INT NOT NULL DEFAULT 0,
    error_count INT NOT NULL DEFAULT 0,
    error_message TEXT DEFAULT '',
    metadata JSONB DEFAULT '{}'::jsonb,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scrape_jobs_status ON scrape_jobs(status);
CREATE INDEX IF NOT EXISTS idx_scrape_jobs_created_at ON scrape_jobs(created_at DESC);

-- 3. Business Leads Table
CREATE TABLE IF NOT EXISTS business_leads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID REFERENCES scrape_jobs(id) ON DELETE SET NULL,
    place_id VARCHAR(255) UNIQUE,
    cid VARCHAR(100),
    name VARCHAR(255) NOT NULL,
    category VARCHAR(255) NOT NULL DEFAULT '',
    subcategories JSONB DEFAULT '[]'::jsonb,
    address TEXT NOT NULL DEFAULT '',
    street VARCHAR(255) DEFAULT '',
    city VARCHAR(100) DEFAULT '',
    state VARCHAR(100) DEFAULT '',
    postal_code VARCHAR(50) DEFAULT '',
    country VARCHAR(100) DEFAULT '',
    latitude DOUBLE PRECISION DEFAULT 0.0,
    longitude DOUBLE PRECISION DEFAULT 0.0,
    rating DOUBLE PRECISION DEFAULT 0.0,
    review_count INT DEFAULT 0,
    price_level VARCHAR(50) DEFAULT '',
    claimed BOOLEAN DEFAULT false,
    business_status VARCHAR(50) DEFAULT 'OPERATIONAL',
    website TEXT DEFAULT '',
    primary_phone VARCHAR(100) DEFAULT '',
    emails JSONB DEFAULT '[]'::jsonb,
    phones JSONB DEFAULT '[]'::jsonb,
    facebook_url TEXT DEFAULT '',
    instagram_url TEXT DEFAULT '',
    linkedin_url TEXT DEFAULT '',
    twitter_url TEXT DEFAULT '',
    youtube_url TEXT DEFAULT '',
    tiktok_url TEXT DEFAULT '',
    google_maps_url TEXT DEFAULT '',
    opening_hours JSONB DEFAULT '{}'::jsonb,
    photos JSONB DEFAULT '[]'::jsonb,
    enrichment_status VARCHAR(50) DEFAULT 'NONE',
    last_scraped_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_enriched_at TIMESTAMPTZ,
    contacted BOOLEAN NOT NULL DEFAULT false,
    contacted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_business_leads_name ON business_leads(name);
CREATE INDEX IF NOT EXISTS idx_business_leads_category ON business_leads(category);
CREATE INDEX IF NOT EXISTS idx_business_leads_city ON business_leads(city);
CREATE INDEX IF NOT EXISTS idx_business_leads_rating ON business_leads(rating DESC);
CREATE INDEX IF NOT EXISTS idx_business_leads_job_id ON business_leads(job_id);
CREATE INDEX IF NOT EXISTS idx_business_leads_enrichment_status ON business_leads(enrichment_status);
CREATE INDEX IF NOT EXISTS idx_business_leads_created_at ON business_leads(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_business_leads_lower_name_city ON business_leads(LOWER(name), LOWER(city));
CREATE INDEX IF NOT EXISTS idx_business_leads_lower_website ON business_leads(LOWER(website));
CREATE INDEX IF NOT EXISTS idx_business_leads_phone ON business_leads(primary_phone);

-- 4. System Settings Table
CREATE TABLE IF NOT EXISTS system_settings (
    key VARCHAR(100) PRIMARY KEY,
    value JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5. Scraper Keywords Table (Stateful Cursor Pagination Registry)
CREATE TABLE IF NOT EXISTS scraper_keywords (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    keyword VARCHAR(255) UNIQUE NOT NULL,
    next_page_token TEXT DEFAULT '',
    current_page INT NOT NULL DEFAULT 1,
    total_leads_scraped INT NOT NULL DEFAULT 0,
    last_scraped_at TIMESTAMPTZ,
    status VARCHAR(50) NOT NULL DEFAULT 'ready',
    is_active BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scraper_keywords_status ON scraper_keywords(status);
CREATE INDEX IF NOT EXISTS idx_scraper_keywords_active ON scraper_keywords(is_active);
CREATE INDEX IF NOT EXISTS idx_scraper_keywords_updated_at ON scraper_keywords(updated_at DESC);

-- 6. Non-destructive Alterations (Ensures existing tables get new columns on startup)
ALTER TABLE business_leads ADD COLUMN IF NOT EXISTS contacted BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE business_leads ADD COLUMN IF NOT EXISTS contacted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_business_leads_contacted ON business_leads(contacted);

-- Lead status (cold / warm / hot), set by the admin on contacted leads
ALTER TABLE business_leads ADD COLUMN IF NOT EXISTS lead_status VARCHAR(10) CHECK (lead_status IN ('cold', 'warm', 'hot'));
UPDATE business_leads SET lead_status = 'cold' WHERE contacted = true AND lead_status IS NULL;
CREATE INDEX IF NOT EXISTS idx_business_leads_lead_status ON business_leads(lead_status);

-- Repair any scalar or non-array jsonb values in business_leads
UPDATE business_leads SET emails = '[]'::jsonb WHERE emails IS NULL OR jsonb_typeof(emails) != 'array';
UPDATE business_leads SET phones = '[]'::jsonb WHERE phones IS NULL OR jsonb_typeof(phones) != 'array';
UPDATE business_leads SET subcategories = '[]'::jsonb WHERE subcategories IS NULL OR jsonb_typeof(subcategories) != 'array';
UPDATE business_leads SET photos = '[]'::jsonb WHERE photos IS NULL OR jsonb_typeof(photos) != 'array';

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

-- Deduplication index on name and address
CREATE INDEX IF NOT EXISTS idx_business_leads_lower_name_address ON business_leads(LOWER(TRIM(name)), LOWER(TRIM(address)));




-- API keys used to be copied into scrape_jobs.metadata in plain text. Remove them.
UPDATE scrape_jobs SET metadata = metadata - 'api_key' WHERE metadata ? 'api_key';

-- Durable job queue: scrape_jobs rows are claimed by workers and survive restarts.
ALTER TABLE scrape_jobs ADD COLUMN IF NOT EXISTS kind VARCHAR(20) NOT NULL DEFAULT 'manual';
ALTER TABLE scrape_jobs ADD COLUMN IF NOT EXISTS keyword_id UUID;
ALTER TABLE scrape_jobs ADD COLUMN IF NOT EXISTS attempts INT NOT NULL DEFAULT 0;
ALTER TABLE scrape_jobs ADD COLUMN IF NOT EXISTS heartbeat_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_scrape_jobs_pending ON scrape_jobs(created_at) WHERE status = 'PENDING';
CREATE INDEX IF NOT EXISTS idx_scrape_jobs_keyword_active ON scrape_jobs(keyword_id) WHERE status IN ('PENDING', 'PROCESSING');

-- Older keyword jobs were stored as plain jobs: mark them as keyword jobs so a requeue runs the right pipeline.
UPDATE scrape_jobs SET kind = 'keyword', keyword_id = (metadata->>'keyword_id')::uuid
WHERE kind = 'manual' AND metadata->>'source' = 'auto_scraper_keyword' AND metadata->>'keyword_id' ~ '^[0-9a-fA-F-]{36}$';
UPDATE scrape_jobs j SET kind = 'keyword', keyword_id = k.id
FROM scraper_keywords k
WHERE j.kind = 'manual' AND j.category = 'Keyword Scrape' AND j.query = k.keyword;
-- Keyword jobs whose keyword was deleted keep keyword_id NULL, and a queued one fails with "Keyword no longer exists".
UPDATE scrape_jobs SET kind = 'keyword'
WHERE kind = 'manual' AND (category = 'Keyword Scrape' OR metadata->>'source' = 'auto_scraper_keyword');

-- Scheduler state lives in the database so restarts don't move or skip runs, and only one instance
-- can start a given run.
CREATE TABLE IF NOT EXISTS scheduler_state (
    id INT PRIMARY KEY CHECK (id = 1),
    next_run_at TIMESTAMPTZ,
    last_run_at TIMESTAMPTZ,
    quota_paused_until TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
INSERT INTO scheduler_state (id) VALUES (1) ON CONFLICT (id) DO NOTHING;
