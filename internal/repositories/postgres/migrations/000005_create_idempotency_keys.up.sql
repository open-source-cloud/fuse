CREATE TABLE IF NOT EXISTS idempotency_keys (
    id              BIGSERIAL       PRIMARY KEY,
    idempotency_key VARCHAR(512)    NOT NULL UNIQUE,
    workflow_id     VARCHAR(36)     NOT NULL,
    expires_at      TIMESTAMPTZ     NOT NULL,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_idempotency_keys_expires_at ON idempotency_keys (expires_at);
