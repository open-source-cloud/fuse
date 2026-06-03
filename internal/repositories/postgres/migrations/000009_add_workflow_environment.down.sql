DROP INDEX IF EXISTS idx_workflows_environment;
ALTER TABLE workflows DROP COLUMN IF EXISTS environment;
