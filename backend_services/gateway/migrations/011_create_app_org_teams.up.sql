CREATE TABLE app_org_teams (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id              UUID NOT NULL REFERENCES app_organizations (id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    created_by_user_id  UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT app_org_teams_org_name_unique UNIQUE (org_id, name)
);

CREATE INDEX idx_app_org_teams_org_id ON app_org_teams (org_id);

CREATE TABLE app_org_team_memberships (
    team_id   UUID NOT NULL REFERENCES app_org_teams (id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    added_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (team_id, user_id)
);

CREATE INDEX idx_app_org_team_memberships_user_id ON app_org_team_memberships (user_id);
