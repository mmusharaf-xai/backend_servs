ALTER TABLE app_org_invites DROP CONSTRAINT app_org_invites_status_check;

ALTER TABLE app_org_invites
    ADD CONSTRAINT app_org_invites_status_check CHECK (status IN ('pending', 'accepted', 'deactive'));
