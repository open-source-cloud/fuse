-- Execution traces: persistable summaries of workflow runs.
-- Steps are stored in a separate 1:N table for proper normalization.

CREATE TABLE execution_traces (
    id              BIGSERIAL       PRIMARY KEY,
    workflow_id     VARCHAR(36)     NOT NULL UNIQUE,
    schema_id       VARCHAR(128)    NOT NULL,
    status          workflow_state  NOT NULL,
    triggered_at    TIMESTAMPTZ     NOT NULL,
    completed_at    TIMESTAMPTZ,
    duration        VARCHAR(32),
    error           VARCHAR(2048),
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_trace_workflow FOREIGN KEY (workflow_id)
        REFERENCES workflows(workflow_id)
);

CREATE INDEX idx_traces_schema ON execution_traces (schema_id, triggered_at DESC);
CREATE INDEX idx_traces_status ON execution_traces (status);

CREATE TABLE execution_trace_steps (
    id                  BIGSERIAL       PRIMARY KEY,
    workflow_id         VARCHAR(36)     NOT NULL,
    exec_id             VARCHAR(64)     NOT NULL,
    thread_id           SMALLINT        NOT NULL DEFAULT 0,
    function_node_id    VARCHAR(128)    NOT NULL,
    started_at          TIMESTAMPTZ     NOT NULL,
    completed_at        TIMESTAMPTZ,
    duration            VARCHAR(32),
    input_ref           VARCHAR(512),
    output_ref          VARCHAR(512),
    status              VARCHAR(32)     NOT NULL,
    attempt             INT             NOT NULL DEFAULT 1,
    error               VARCHAR(2048),

    CONSTRAINT fk_trace_step_workflow FOREIGN KEY (workflow_id)
        REFERENCES execution_traces(workflow_id) ON DELETE CASCADE
);

CREATE INDEX idx_trace_steps_workflow ON execution_trace_steps (workflow_id);
