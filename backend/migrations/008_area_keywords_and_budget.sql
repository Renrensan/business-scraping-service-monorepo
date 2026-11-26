-- Migration 008: areas, area keywords, keyword cells and Places request budget.
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
