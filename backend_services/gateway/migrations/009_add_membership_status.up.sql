ALTER TABLE app_org_memberships
    ADD COLUMN status TEXT NOT NULL DEFAULT 'active';

ALTER TABLE app_org_memberships
    ADD CONSTRAINT app_org_memberships_status_check CHECK (status IN ('active', 'deactive'));

CREATE INDEX idx_app_org_memberships_org_status ON app_org_memberships (org_id, status);
