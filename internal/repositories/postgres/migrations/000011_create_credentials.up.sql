-- ADR-0031 Phase 2 (Option B): typed, centrally-managed credential metadata.
-- Field VALUES are NOT stored here; they live in the secrets table at name
-- cred/<id>/<field>, per environment (reusing the encrypted secret store).
-- NOTE: numbered 000011 to follow the environments table (000010); this migration
-- must be applied after that one when both land on main.

CREATE TABLE credentials (
    id          VARCHAR(128) PRIMARY KEY,
    type        VARCHAR(128) NOT NULL,
    description VARCHAR(512) NOT NULL DEFAULT '',
    fields      TEXT[]       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
