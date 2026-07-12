CREATE TABLE personal_api_keys (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label        TEXT NOT NULL,
    prefix       TEXT NOT NULL,
    secure_value TEXT NOT NULL UNIQUE,
    last_used_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_personal_api_keys_user_id ON personal_api_keys (user_id);
CREATE INDEX idx_personal_api_keys_secure_value ON personal_api_keys (secure_value);
