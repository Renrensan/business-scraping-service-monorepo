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

-- Areas, term+type+area keywords, area cells and Places request budget.
CREATE TABLE IF NOT EXISTS areas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    low_lat DOUBLE PRECISION NOT NULL,
    low_lng DOUBLE PRECISION NOT NULL,
    high_lat DOUBLE PRECISION NOT NULL,
    high_lng DOUBLE PRECISION NOT NULL,
    source VARCHAR(20) NOT NULL DEFAULT 'manual',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (low_lat < high_lat AND low_lng < high_lng)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_areas_lower_name ON areas(LOWER(name));

-- Approximate rectangles around common Bali areas (center +- a few km). Replace with a looked-up or manual area if needed.
INSERT INTO areas (name, low_lat, low_lng, high_lat, high_lng, source) VALUES
('Amed', -8.3750, 115.6177, -8.3210, 115.6723, 'preset'),
('Badung', -8.7061, 115.0527, -8.4539, 115.3073, 'preset'),
('Berawa', -8.6755, 115.1264, -8.6485, 115.1536, 'preset'),
('Bingin', -8.8185, 115.0994, -8.7915, 115.1266, 'preset'),
('Candidasa', -8.5325, 115.5453, -8.4875, 115.5907, 'preset'),
('Canggu', -8.6703, 115.1158, -8.6253, 115.1612, 'preset'),
('Denpasar', -8.7041, 115.1615, -8.5959, 115.2705, 'preset'),
('Jatiluwih', -8.3970, 115.1067, -8.3430, 115.1613, 'preset'),
('Jimbaran', -8.8020, 115.1437, -8.7480, 115.1983, 'preset'),
('Kedungu', -8.6230, 115.0698, -8.5870, 115.1062, 'preset'),
('Keramas', -8.6040, 115.3258, -8.5680, 115.3622, 'preset'),
('Kerobokan', -8.6740, 115.1518, -8.6380, 115.1882, 'preset'),
('Kintamani', -8.3021, 115.2695, -8.1939, 115.3785, 'preset'),
('Kuta', -8.7360, 115.1548, -8.7000, 115.1912, 'preset'),
('Legian', -8.7181, 115.1564, -8.6911, 115.1836, 'preset'),
('Lovina', -8.1940, 114.9896, -8.1220, 115.0624, 'preset'),
('Munduk', -8.2930, 115.0477, -8.2390, 115.1023, 'preset'),
('Nusa Dua', -8.8270, 115.2027, -8.7730, 115.2573, 'preset'),
('Nusa Lembongan', -8.7115, 115.4182, -8.6485, 115.4818, 'preset'),
('Nusa Penida', -8.8356, 115.4353, -8.6194, 115.6535, 'preset'),
('Payangan', -8.4560, 115.2036, -8.3840, 115.2764, 'preset'),
('Pecatu', -8.8430, 115.0897, -8.7890, 115.1443, 'preset'),
('Pemuteran', -8.1720, 114.6447, -8.1180, 114.6993, 'preset'),
('Pererenan', -8.6545, 115.1084, -8.6275, 115.1356, 'preset'),
('Sanur', -8.7150, 115.2347, -8.6610, 115.2893, 'preset'),
('Seminyak', -8.7093, 115.1500, -8.6733, 115.1864, 'preset'),
('Sidemen', -8.5070, 115.4127, -8.4530, 115.4673, 'preset'),
('Tabanan', -8.6121, 115.0523, -8.4679, 115.1977, 'preset'),
('Tanjung Benoa', -8.7780, 115.2038, -8.7420, 115.2402, 'preset'),
('Tegallalang', -8.4570, 115.2507, -8.4030, 115.3053, 'preset'),
('Tulamben', -8.3005, 115.5703, -8.2555, 115.6157, 'preset'),
('Ubud', -8.5429, 115.2261, -8.4709, 115.2989, 'preset'),
('Uluwatu', -8.8560, 115.0607, -8.8020, 115.1153, 'preset'),
('Umalas', -8.6775, 115.1444, -8.6505, 115.1716, 'preset')
ON CONFLICT DO NOTHING;

ALTER TABLE scraper_keywords ADD COLUMN IF NOT EXISTS mode VARCHAR(10) NOT NULL DEFAULT 'plain';
ALTER TABLE scraper_keywords ADD COLUMN IF NOT EXISTS term TEXT NOT NULL DEFAULT '';
ALTER TABLE scraper_keywords ADD COLUMN IF NOT EXISTS included_type VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE scraper_keywords ADD COLUMN IF NOT EXISTS area_id UUID REFERENCES areas(id) ON DELETE RESTRICT;
ALTER TABLE scraper_keywords ADD COLUMN IF NOT EXISTS new_places_last_run INT NOT NULL DEFAULT 0;
ALTER TABLE scraper_keywords ADD COLUMN IF NOT EXISTS empty_runs INT NOT NULL DEFAULT 0;
ALTER TABLE scraper_keywords ADD COLUMN IF NOT EXISTS next_eligible_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_scraper_keywords_area ON scraper_keywords(area_id);

CREATE TABLE IF NOT EXISTS keyword_cells (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    keyword_id UUID NOT NULL REFERENCES scraper_keywords(id) ON DELETE CASCADE,
    depth INT NOT NULL DEFAULT 0,
    low_lat DOUBLE PRECISION NOT NULL,
    low_lng DOUBLE PRECISION NOT NULL,
    high_lat DOUBLE PRECISION NOT NULL,
    high_lng DOUBLE PRECISION NOT NULL,
    status VARCHAR(10) NOT NULL DEFAULT 'pending',
    next_page_token TEXT NOT NULL DEFAULT '',
    pages_fetched INT NOT NULL DEFAULT 0,
    results INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_keyword_cells_keyword ON keyword_cells(keyword_id, status);

-- Places requests per Pacific-time day (Google's quota day), for the daily budget.
CREATE TABLE IF NOT EXISTS places_usage (
    day DATE PRIMARY KEY,
    requests INT NOT NULL DEFAULT 0
);
