-- ADR-0031 Phase 3: first-class environments registry (dev/staging/prod).
-- Workflow executions reference an environment to scope secret resolution. The 'default'
-- environment is always present and is what triggers fall back to.

CREATE TABLE environments (
    id          BIGSERIAL    PRIMARY KEY,
    name        VARCHAR(128) NOT NULL UNIQUE,
    description VARCHAR(512) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

INSERT INTO environments (name, description)
    VALUES ('default', 'Default environment')
    ON CONFLICT (name) DO NOTHING;
