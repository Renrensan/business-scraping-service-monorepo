-- Migration 004: Add admin-assigned lead status (cold / warm / hot) to business_leads
--   cold = contacted, no response yet
--   warm = they responded
--   hot  = they look interested in the product
-- NULL means the lead has not been contacted. Existing contacted leads start as cold.

ALTER TABLE business_leads
  ADD COLUMN IF NOT EXISTS lead_status VARCHAR(10) CHECK (lead_status IN ('cold', 'warm', 'hot'));

UPDATE business_leads SET lead_status = 'cold' WHERE contacted = true AND lead_status IS NULL;

CREATE INDEX IF NOT EXISTS idx_business_leads_lead_status ON business_leads(lead_status);
