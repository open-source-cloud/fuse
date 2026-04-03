-- Reverse of 000001_create_tables.up.sql
-- Drop tables in reverse dependency order, then types.

DROP TABLE IF EXISTS node_heartbeats;
DROP TABLE IF EXISTS package_functions;
DROP TABLE IF EXISTS package_tags;
DROP TABLE IF EXISTS packages;
DROP TABLE IF EXISTS awakeables;
DROP TABLE IF EXISTS graph_schema_nodes;
DROP TABLE IF EXISTS graph_schema_metadata;
DROP TABLE IF EXISTS graph_schema_tags;
DROP TABLE IF EXISTS graph_schemas;
DROP TABLE IF EXISTS journal_entries;
DROP TABLE IF EXISTS sub_workflow_refs;
DROP TABLE IF EXISTS workflows;

DROP TYPE IF EXISTS transport_type;
DROP TYPE IF EXISTS awakeable_status;
DROP TYPE IF EXISTS journal_entry_type;
DROP TYPE IF EXISTS workflow_state;
