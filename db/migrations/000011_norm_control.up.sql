BEGIN;

-- =========================
-- Norm control checks (1 check == 1 submission)
-- =========================
CREATE TABLE IF NOT EXISTS norm_control_checks (
                                                   submission_id VARCHAR(36) PRIMARY KEY REFERENCES admin_submissions(id) ON DELETE CASCADE,

    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    team_id BIGINT REFERENCES teams(id) ON DELETE SET NULL,
    step_id BIGINT REFERENCES states(id) ON DELETE SET NULL,

    primary_file_id VARCHAR(36) REFERENCES file_metadata(id) ON DELETE SET NULL,
    file_ids JSONB NOT NULL DEFAULT '[]'::jsonb,

    document_version INT NOT NULL DEFAULT 1,

    checker_id BIGINT REFERENCES users(id) ON DELETE SET NULL,

    status VARCHAR(50) NOT NULL DEFAULT 'submitted', -- submitted, in_review, returned, approved
    overall_comment TEXT,

    started_at TIMESTAMPTZ,
    reviewed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_norm_checks_project ON norm_control_checks(project_id);
CREATE INDEX IF NOT EXISTS idx_norm_checks_status ON norm_control_checks(status);
CREATE INDEX IF NOT EXISTS idx_norm_checks_checker ON norm_control_checks(checker_id);
CREATE INDEX IF NOT EXISTS idx_norm_checks_step ON norm_control_checks(step_id);

-- =========================
-- Issues
-- =========================
CREATE TABLE IF NOT EXISTS norm_control_issues (
                                                   id UUID PRIMARY KEY,
                                                   submission_id VARCHAR(36) NOT NULL REFERENCES norm_control_checks(submission_id) ON DELETE CASCADE,

    category VARCHAR(100) NOT NULL, -- formatting, structure, references, tables, ...
    severity VARCHAR(50) NOT NULL,  -- critical, major, minor
    description TEXT NOT NULL,

    page_number INT,
    line_number INT,

    is_resolved BOOLEAN NOT NULL DEFAULT FALSE,
    resolved_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_norm_issues_submission ON norm_control_issues(submission_id);
CREATE INDEX IF NOT EXISTS idx_norm_issues_severity ON norm_control_issues(severity);
CREATE INDEX IF NOT EXISTS idx_norm_issues_resolved ON norm_control_issues(is_resolved);

-- =========================
-- History (audit)
-- =========================
CREATE TABLE IF NOT EXISTS norm_control_history (
                                                    id BIGSERIAL PRIMARY KEY,
                                                    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    submission_id VARCHAR(36) REFERENCES admin_submissions(id) ON DELETE SET NULL,

    action VARCHAR(100) NOT NULL, -- submitted, started_review, issue_added, issue_updated, issue_deleted, returned, approved, resubmitted
    actor_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    comment TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_norm_history_project ON norm_control_history(project_id);
CREATE INDEX IF NOT EXISTS idx_norm_history_submission ON norm_control_history(submission_id);

-- =========================
-- Checklists
-- =========================
CREATE TABLE IF NOT EXISTS norm_control_checklists (
                                                       id UUID PRIMARY KEY,
                                                       name VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL,
    items JSONB NOT NULL,

    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_norm_checklists_active ON norm_control_checklists(is_active);

COMMIT;
