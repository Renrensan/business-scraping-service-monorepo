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

-- Repair any scalar or non-array jsonb values in business_leads
UPDATE business_leads SET emails = '[]'::jsonb WHERE emails IS NULL OR jsonb_typeof(emails) != 'array';
UPDATE business_leads SET phones = '[]'::jsonb WHERE phones IS NULL OR jsonb_typeof(phones) != 'array';
UPDATE business_leads SET subcategories = '[]'::jsonb WHERE subcategories IS NULL OR jsonb_typeof(subcategories) != 'array';
UPDATE business_leads SET photos = '[]'::jsonb WHERE photos IS NULL OR jsonb_typeof(photos) != 'array';

-- Deduplication index on name and address
CREATE INDEX IF NOT EXISTS idx_business_leads_lower_name_address ON business_leads(LOWER(TRIM(name)), LOWER(TRIM(address)));



