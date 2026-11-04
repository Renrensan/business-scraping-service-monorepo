-- Migration 002: Add "contacted" tracking to business_leads
-- Safe for existing data: DEFAULT false backfills all existing rows instantly (metadata-only op in Postgres 12+)

ALTER TABLE business_leads
  ADD COLUMN IF NOT EXISTS contacted BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS contacted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_business_leads_contacted ON business_leads(contacted);
