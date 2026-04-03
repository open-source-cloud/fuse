-- Phase 2.5: Core tables for durable persistence
-- All tables follow 3NF with BIGSERIAL PKs, PostgreSQL ENUMs, and VARCHAR(n).

-- ============================================================================
-- ENUM TYPES
-- ============================================================================

CREATE TYPE workflow_state AS ENUM (
    'untriggered', 'running', 'sleeping', 'finished', 'error', 'cancelled'
);

CREATE TYPE journal_entry_type AS ENUM (
    'step:started', 'step:completed', 'step:failed', 'step:retrying',
    'thread:created', 'thread:finished',
    'state:changed',
    'sleep:started', 'sleep:completed',
    'awakeable:created', 'awakeable:resolved',
    'subworkflow:started', 'subworkflow:completed'
);

CREATE TYPE awakeable_status AS ENUM (
    'pending', 'resolved', 'timed_out', 'cancelled'
);

CREATE TYPE transport_type AS ENUM (
    'internal', 'http', 'grpc'
);

-- ============================================================================
-- CORE TABLES
-- ============================================================================

-- Workflow execution instances
CREATE TABLE workflows (
    id              BIGSERIAL       PRIMARY KEY,
    workflow_id     VARCHAR(36)     NOT NULL UNIQUE,
    schema_id       VARCHAR(128)    NOT NULL,
    state           workflow_state  NOT NULL DEFAULT 'untriggered',
    output_ref      VARCHAR(512),
    claimed_by      VARCHAR(128),
    claimed_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workflows_state ON workflows (state);
CREATE INDEX idx_workflows_claimed ON workflows (claimed_by, state) WHERE claimed_by IS NOT NULL;
CREATE INDEX idx_workflows_schema ON workflows (schema_id);
CREATE INDEX idx_workflows_created ON workflows (created_at);

-- Sub-workflow references
CREATE TABLE sub_workflow_refs (
    id                  BIGSERIAL       PRIMARY KEY,
    child_workflow_id   VARCHAR(36)     NOT NULL UNIQUE,
    parent_workflow_id  VARCHAR(36)     NOT NULL,
    parent_thread_id    SMALLINT        NOT NULL,
    parent_exec_id      VARCHAR(64)     NOT NULL,
    child_schema_id     VARCHAR(128)    NOT NULL,
    async               BOOLEAN         NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_subwf_parent FOREIGN KEY (parent_workflow_id)
        REFERENCES workflows(workflow_id)
);

CREATE INDEX idx_subwf_parent ON sub_workflow_refs (parent_workflow_id);

-- Execution journal (append-only)
CREATE TABLE journal_entries (
    id                  BIGSERIAL           PRIMARY KEY,
    workflow_id         VARCHAR(36)         NOT NULL,
    sequence            BIGINT              NOT NULL,
    entry_type          journal_entry_type  NOT NULL,
    thread_id           SMALLINT            NOT NULL DEFAULT 0,
    function_node_id    VARCHAR(128),
    exec_id             VARCHAR(64),
    state               workflow_state,
    parent_threads      SMALLINT[],
    input_ref           VARCHAR(512),
    result_ref          VARCHAR(512),
    data_ref            VARCHAR(512),
    created_at          TIMESTAMPTZ         NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_journal_workflow FOREIGN KEY (workflow_id)
        REFERENCES workflows(workflow_id)
);

CREATE UNIQUE INDEX idx_journal_wf_seq ON journal_entries (workflow_id, sequence);
CREATE INDEX idx_journal_wf_type ON journal_entries (workflow_id, entry_type);
CREATE INDEX idx_journal_created ON journal_entries (created_at);

-- Graph schemas (workflow definitions)
CREATE TABLE graph_schemas (
    id                  BIGSERIAL       PRIMARY KEY,
    schema_id           VARCHAR(128)    NOT NULL UNIQUE,
    name                VARCHAR(100)    NOT NULL,
    timeout_total_ns    BIGINT,
    definition_ref      VARCHAR(512)    NOT NULL,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE TABLE graph_schema_tags (
    id          BIGSERIAL       PRIMARY KEY,
    schema_id   VARCHAR(128)    NOT NULL,
    key         VARCHAR(64)     NOT NULL,
    value       VARCHAR(256)    NOT NULL,

    CONSTRAINT fk_schema_tag FOREIGN KEY (schema_id)
        REFERENCES graph_schemas(schema_id) ON DELETE CASCADE,
    CONSTRAINT uq_schema_tag UNIQUE (schema_id, key)
);

CREATE TABLE graph_schema_metadata (
    id          BIGSERIAL       PRIMARY KEY,
    schema_id   VARCHAR(128)    NOT NULL,
    key         VARCHAR(64)     NOT NULL,
    value       VARCHAR(1024)   NOT NULL,

    CONSTRAINT fk_schema_meta FOREIGN KEY (schema_id)
        REFERENCES graph_schemas(schema_id) ON DELETE CASCADE,
    CONSTRAINT uq_schema_meta UNIQUE (schema_id, key)
);

CREATE TABLE graph_schema_nodes (
    id              BIGSERIAL       PRIMARY KEY,
    schema_id       VARCHAR(128)    NOT NULL,
    node_id         VARCHAR(128)    NOT NULL,
    function_ref    VARCHAR(256)    NOT NULL,

    CONSTRAINT fk_schema_node FOREIGN KEY (schema_id)
        REFERENCES graph_schemas(schema_id) ON DELETE CASCADE,
    CONSTRAINT uq_schema_node UNIQUE (schema_id, node_id)
);

CREATE INDEX idx_schema_nodes_func ON graph_schema_nodes (function_ref);

-- Awakeables (durable promises)
CREATE TABLE awakeables (
    id              BIGSERIAL           PRIMARY KEY,
    awakeable_id    VARCHAR(36)         NOT NULL UNIQUE,
    workflow_id     VARCHAR(36)         NOT NULL,
    exec_id         VARCHAR(64)         NOT NULL,
    thread_id       SMALLINT            NOT NULL,
    status          awakeable_status    NOT NULL DEFAULT 'pending',
    timeout_ns      BIGINT,
    deadline_at     TIMESTAMPTZ,
    result_ref      VARCHAR(512),
    created_at      TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ         NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_awakeable_workflow FOREIGN KEY (workflow_id)
        REFERENCES workflows(workflow_id)
);

CREATE INDEX idx_awakeables_wf_status ON awakeables (workflow_id, status);

-- Packages (function registries)
CREATE TABLE packages (
    id              BIGSERIAL       PRIMARY KEY,
    package_id      VARCHAR(128)    NOT NULL UNIQUE,
    definition_ref  VARCHAR(512)    NOT NULL,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE TABLE package_tags (
    id          BIGSERIAL       PRIMARY KEY,
    package_id  VARCHAR(128)    NOT NULL,
    key         VARCHAR(64)     NOT NULL,
    value       VARCHAR(256)    NOT NULL,

    CONSTRAINT fk_pkg_tag FOREIGN KEY (package_id)
        REFERENCES packages(package_id) ON DELETE CASCADE,
    CONSTRAINT uq_pkg_tag UNIQUE (package_id, key)
);

CREATE TABLE package_functions (
    id              BIGSERIAL       PRIMARY KEY,
    package_id      VARCHAR(128)    NOT NULL,
    function_id     VARCHAR(128)    NOT NULL,
    transport       transport_type  NOT NULL,

    CONSTRAINT fk_pkg_func FOREIGN KEY (package_id)
        REFERENCES packages(package_id) ON DELETE CASCADE,
    CONSTRAINT uq_pkg_func UNIQUE (package_id, function_id)
);

-- Node heartbeats (HA lease management)
CREATE TABLE node_heartbeats (
    id          BIGSERIAL       PRIMARY KEY,
    node_id     VARCHAR(128)    NOT NULL UNIQUE,
    host        VARCHAR(256),
    port        SMALLINT,
    started_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    last_seen   TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);
