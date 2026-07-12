CREATE TABLE apps (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug        TEXT NOT NULL,
    name        TEXT NOT NULL,
    tagline     TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    icon        TEXT NOT NULL DEFAULT '',
    category    TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'available',
    sort_order  INT  NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT apps_slug_unique UNIQUE (slug)
);

CREATE INDEX idx_apps_sort_order_id ON apps (sort_order, id);

INSERT INTO apps (slug, name, tagline, description, icon, category, status, sort_order)
VALUES ('surveillance-pro', 'SurveillancePro',
  'AI-powered surveillance',
  'An app used to do surveillance using AI.',
  'scan-eye', 'Security', 'available', 0);
