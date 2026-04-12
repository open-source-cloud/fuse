-- Phase 4.2: Workflow Versioning
-- Each PUT to a schema creates a new immutable version instead of overwriting.
-- Existing schemas are back-filled as version 1 on migration.

-- Add active_version tracking to graph_schemas
ALTER TABLE graph_schemas
    ADD COLUMN active_version INTEGER NOT NULL DEFAULT 1;

-- Immutable version snapshots for each schema
CREATE TABLE graph_schema_versions (
    id              BIGSERIAL       PRIMARY KEY,
    schema_id       VARCHAR(128)    NOT NULL,
    version         INTEGER         NOT NULL,
    definition_ref  VARCHAR(512)    NOT NULL,
    created_by      VARCHAR(256),
    comment         VARCHAR(1024),
    is_active       BOOLEAN         NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_schema_version FOREIGN KEY (schema_id)
        REFERENCES graph_schemas(schema_id) ON DELETE CASCADE,
    CONSTRAINT uq_schema_version UNIQUE (schema_id, version)
);

CREATE INDEX idx_schema_versions_schema ON graph_schema_versions (schema_id);
CREATE INDEX idx_schema_versions_active ON graph_schema_versions (schema_id, is_active) WHERE is_active = TRUE;

-- Back-fill: existing schemas become version 1 (copies their current definition_ref)
INSERT INTO graph_schema_versions (schema_id, version, definition_ref, comment, is_active, created_at)
SELECT schema_id, 1, definition_ref, 'Initial version (migrated)', TRUE, created_at
FROM graph_schemas;
