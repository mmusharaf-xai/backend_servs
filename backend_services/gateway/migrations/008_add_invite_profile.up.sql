ALTER TABLE app_org_invites ADD COLUMN first_name TEXT NOT NULL DEFAULT '';
ALTER TABLE app_org_invites ADD COLUMN last_name TEXT NOT NULL DEFAULT '';
ALTER TABLE app_org_invites ADD COLUMN phone TEXT;
