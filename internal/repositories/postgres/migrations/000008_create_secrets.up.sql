-- ADR-0031 Phase 1: encrypted-at-rest secret store.
-- Values are AES-256-GCM encrypted by the application before being stored here;
-- the encrypted_value column only ever holds ciphertext (nonce-prefixed).

CREATE TABLE secrets (
    id              BIGSERIAL    PRIMARY KEY,
    environment     VARCHAR(128) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    encrypted_value BYTEA        NOT NULL,
    source          VARCHAR(50)  NOT NULL DEFAULT 'manual',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_secrets_env_name UNIQUE (environment, name)
);

CREATE INDEX idx_secrets_environment ON secrets (environment);
