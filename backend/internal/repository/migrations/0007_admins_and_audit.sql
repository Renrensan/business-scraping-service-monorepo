-- Multiple admins: who contacted a lead, last login, and an audit log of every admin action.

ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;

-- The admin who marked the lead contacted. Existing contacted leads stay unknown (NULL).
ALTER TABLE business_leads ADD COLUMN IF NOT EXISTS contacted_by UUID REFERENCES admin_users(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_business_leads_contacted_by ON business_leads(contacted_by);

-- One row per admin action (every change made through the API, plus logins and logouts).
-- admin_name is a snapshot, so entries stay readable if the account is renamed or removed.
CREATE TABLE IF NOT EXISTS audit_log (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    admin_id UUID REFERENCES admin_users(id) ON DELETE SET NULL,
    admin_name TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    entity_type TEXT NOT NULL DEFAULT '',
    entity_id TEXT NOT NULL DEFAULT '',
    method TEXT NOT NULL DEFAULT '',
    route TEXT NOT NULL DEFAULT '',
    status INT NOT NULL DEFAULT 0,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    ip TEXT NOT NULL DEFAULT '',
    request_id TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_audit_log_created ON audit_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_entity ON audit_log(entity_type, entity_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_admin ON audit_log(admin_id, created_at DESC);
