-- Rollback Phase 4.2: Workflow Versioning

DROP TABLE IF EXISTS graph_schema_versions;

ALTER TABLE graph_schemas
    DROP COLUMN IF EXISTS active_version;
