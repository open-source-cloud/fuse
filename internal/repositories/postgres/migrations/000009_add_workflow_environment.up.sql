-- ADR-0031 Phase 3: environment becomes a per-execution scoping dimension.
-- The environment chosen at trigger time is persisted here so actor restart / journal
-- replay resolves secrets against the same environment. NOT NULL DEFAULT 'default'
-- backfills in-flight workflows created before this migration.

ALTER TABLE workflows ADD COLUMN environment VARCHAR(128) NOT NULL DEFAULT 'default';

CREATE INDEX idx_workflows_environment ON workflows (environment);
