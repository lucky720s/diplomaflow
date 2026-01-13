-- workflow_action_runs: идемпотентность и учёт выполнений POST actions
CREATE TABLE IF NOT EXISTS workflow_action_runs (
                                                    id BIGSERIAL PRIMARY KEY,
                                                    dedup_key TEXT NOT NULL UNIQUE,

                                                    event_type VARCHAR(100) NOT NULL,
    topic VARCHAR(255) NOT NULL,

    project_id BIGINT NOT NULL,
    state_id BIGINT,
    transition_id BIGINT,
    trigger VARCHAR(50) NOT NULL,
    action_id BIGINT NOT NULL,

    status VARCHAR(30) NOT NULL DEFAULT 'running', -- running, succeeded, failed
    attempts INT NOT NULL DEFAULT 1,
    last_error TEXT,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    succeeded_at TIMESTAMP WITH TIME ZONE
                             );

CREATE INDEX IF NOT EXISTS idx_workflow_action_runs_status ON workflow_action_runs(status, created_at);
CREATE INDEX IF NOT EXISTS idx_workflow_action_runs_project ON workflow_action_runs(project_id, created_at);
