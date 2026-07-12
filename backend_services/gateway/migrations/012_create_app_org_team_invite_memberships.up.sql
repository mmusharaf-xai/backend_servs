CREATE TABLE app_org_team_invite_memberships (
    team_id    UUID NOT NULL REFERENCES app_org_teams (id) ON DELETE CASCADE,
    invite_id  UUID NOT NULL REFERENCES app_org_invites (id) ON DELETE CASCADE,
    added_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (team_id, invite_id)
);

CREATE INDEX idx_app_org_team_invite_memberships_invite_id
    ON app_org_team_invite_memberships (invite_id);
