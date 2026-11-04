-- Migration 003: Repair scalar jsonb fields in business_leads and add deduplication index
-- 1. Repair any scalar or non-array jsonb values in business_leads (prevents SQLSTATE 22023)
UPDATE business_leads SET emails = '[]'::jsonb WHERE emails IS NULL OR jsonb_typeof(emails) != 'array';
UPDATE business_leads SET phones = '[]'::jsonb WHERE phones IS NULL OR jsonb_typeof(phones) != 'array';
UPDATE business_leads SET subcategories = '[]'::jsonb WHERE subcategories IS NULL OR jsonb_typeof(subcategories) != 'array';
UPDATE business_leads SET photos = '[]'::jsonb WHERE photos IS NULL OR jsonb_typeof(photos) != 'array';

-- 2. Deduplication index on name and address for resilient upserts
CREATE INDEX IF NOT EXISTS idx_business_leads_lower_name_address ON business_leads(LOWER(TRIM(name)), LOWER(TRIM(address)));
