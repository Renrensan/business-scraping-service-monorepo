-- Lead scoring: admin-defined scoring profiles (GuestChatPro first), per-lead group type/name, and
-- stored scores and tiers per lead per profile.

-- Group type and name: set by hand (NULL = use the suggestion from the lead's name).
ALTER TABLE business_leads ADD COLUMN IF NOT EXISTS group_type VARCHAR(20);
ALTER TABLE business_leads ADD COLUMN IF NOT EXISTS group_name TEXT;
ALTER TABLE business_leads ADD COLUMN IF NOT EXISTS suggested_group_type VARCHAR(20) NOT NULL DEFAULT 'independent';
ALTER TABLE business_leads ADD COLUMN IF NOT EXISTS suggested_group_name TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS scoring_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    applies_to_types JSONB NOT NULL DEFAULT '[]'::jsonb,
    config JSONB NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_scoring_profiles_lower_name ON scoring_profiles(LOWER(name));

CREATE TABLE IF NOT EXISTS lead_scores (
    lead_id UUID NOT NULL REFERENCES business_leads(id) ON DELETE CASCADE,
    profile_id UUID NOT NULL REFERENCES scoring_profiles(id) ON DELETE CASCADE,
    score INT NOT NULL,
    raw_score DOUBLE PRECISION NOT NULL,
    computed_tier VARCHAR(10) NOT NULL,
    tier_override VARCHAR(10) CHECK (tier_override IN ('hot', 'good', 'low', 'skip')),
    tier VARCHAR(10) GENERATED ALWAYS AS (COALESCE(tier_override, computed_tier)) STORED,
    note TEXT NOT NULL DEFAULT '',
    area_name TEXT NOT NULL DEFAULT '',
    breakdown JSONB NOT NULL DEFAULT '{}'::jsonb,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (lead_id, profile_id)
);
CREATE INDEX IF NOT EXISTS idx_lead_scores_profile_tier ON lead_scores(profile_id, tier, score DESC);

-- GuestChatPro, from "GuestChatPro – Approach Priority Score Factors".
INSERT INTO scoring_profiles (name, description, applies_to_types, config) VALUES (
    'GuestChatPro',
    'Cold-approach priority for stays: guest volume (reviews), rating, active online channels and group deals.',
    '["Lodging"]',
    '{
      "reviews_weight": 10, "rating_baseline": 4, "rating_weight": 10,
      "channel_points": 4, "channel_cap": 20,
      "channels": ["email", "website", "instagram", "facebook", "tiktok"],
      "group_bonuses": {"group": 15, "small_group": 8, "independent": 0, "restaurant_stay": -5, "chain": -15, "skip": -30},
      "chain_forces_low": true, "ignore_junk_contacts": true,
      "default_cutoff": {"hot": 50, "good": 38},
      "area_cutoffs": {
        "Ubud": {"hot": 61, "good": 45}, "Kintamani": {"hot": 43, "good": 31}, "Jatiluwih": {"hot": 48, "good": 31},
        "Tabanan": {"hot": 47, "good": 31}, "Pecatu": {"hot": 50, "good": 38}, "Kerobokan": {"hot": 52, "good": 38},
        "Umalas": {"hot": 45, "good": 32}, "Berawa": {"hot": 52, "good": 38}, "Legian": {"hot": 64, "good": 44},
        "Kuta": {"hot": 62, "good": 45}, "Denpasar": {"hot": 50, "good": 38}, "Sanur": {"hot": 56, "good": 42},
        "Uluwatu": {"hot": 40, "good": 30}, "Nusa Dua": {"hot": 47, "good": 34}, "Pererenan": {"hot": 46, "good": 37},
        "Canggu": {"hot": 50, "good": 40}, "Seminyak": {"hot": 59, "good": 45}, "Jimbaran": {"hot": 53, "good": 42},
        "Badung": {"hot": 49, "good": 31}
      }
    }'
) ON CONFLICT DO NOTHING;
