CREATE TABLE app_organizations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_slug            TEXT NOT NULL REFERENCES apps (slug) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    created_by_user_id  UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX app_orgs_user_name_unique ON app_organizations (app_slug, created_by_user_id, lower(name));

CREATE INDEX idx_app_orgs_app_slug ON app_organizations (app_slug);

CREATE TABLE app_org_memberships (
    org_id    UUID NOT NULL REFERENCES app_organizations (id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role      TEXT NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, user_id),
    CONSTRAINT app_org_memberships_role_check CHECK (role IN ('owner', 'member'))
);

CREATE INDEX idx_app_org_memberships_user ON app_org_memberships (user_id);

CREATE TABLE app_org_invites (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id              UUID NOT NULL REFERENCES app_organizations (id) ON DELETE CASCADE,
    email               TEXT NOT NULL,
    invited_by_user_id  UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    status              TEXT NOT NULL DEFAULT 'pending',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    accepted_at         TIMESTAMPTZ,
    CONSTRAINT app_org_invites_status_check CHECK (status IN ('pending', 'accepted'))
);

CREATE UNIQUE INDEX app_org_invites_org_email_pending_unique
    ON app_org_invites (org_id, lower(email))
    WHERE status = 'pending';

CREATE INDEX idx_app_org_invites_email_pending ON app_org_invites (lower(email)) WHERE status = 'pending';
