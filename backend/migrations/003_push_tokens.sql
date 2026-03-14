-- Push notification device tokens
CREATE TABLE IF NOT EXISTS push_tokens (
    user_id    TEXT        NOT NULL PRIMARY KEY,
    token      TEXT        NOT NULL,
    platform   TEXT        NOT NULL DEFAULT 'unknown',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS push_tokens_token_idx ON push_tokens (token);
