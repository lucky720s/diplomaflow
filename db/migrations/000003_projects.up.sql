-- =============================================
-- Migration: 000003_projects
-- Description: Projects and related tables
-- =============================================

-- Projects
CREATE TABLE IF NOT EXISTS projects (
                                        id BIGSERIAL PRIMARY KEY,
                                        title VARCHAR(255) NOT NULL,
    description TEXT,
    student_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    university_id BIGINT NOT NULL REFERENCES universities(id) ON DELETE CASCADE,
    department_id BIGINT REFERENCES departments(id) ON DELETE SET NULL,
    team_id BIGINT REFERENCES teams(id) ON DELETE SET NULL,
    workflow_id BIGINT NOT NULL REFERENCES workflows(id) ON DELETE RESTRICT,
    workflow_version INT DEFAULT 1,
    workflow_name VARCHAR(255),

    -- Current state
    current_state_id BIGINT REFERENCES states(id),
    current_state_name VARCHAR(255),

    -- Status
    status VARCHAR(50) DEFAULT 'active',

    -- Data storage
    data JSONB DEFAULT '{}',

    -- Deadlines
    deadline_at TIMESTAMP WITH TIME ZONE,
    deadline_processed BOOLEAN DEFAULT FALSE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_projects_student_id ON projects(student_id);
CREATE INDEX IF NOT EXISTS idx_projects_department_id ON projects(department_id);
CREATE INDEX IF NOT EXISTS idx_projects_workflow_id ON projects(workflow_id);
CREATE INDEX IF NOT EXISTS idx_projects_current_state ON projects(current_state_id);
CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
CREATE INDEX IF NOT EXISTS idx_projects_deadline ON projects(deadline_at)
    WHERE deadline_processed = false AND status = 'active';

-- Add FK from teams to projects
ALTER TABLE teams
DROP CONSTRAINT IF EXISTS fk_teams_project,
    ADD CONSTRAINT fk_teams_project
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL;

-- Add FK from file_metadata to projects
ALTER TABLE file_metadata
DROP CONSTRAINT IF EXISTS fk_file_metadata_project,
    ADD CONSTRAINT fk_file_metadata_project
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL;

-- State History
CREATE TABLE IF NOT EXISTS state_histories (
                                               id BIGSERIAL PRIMARY KEY,
                                               project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    from_state_id BIGINT REFERENCES states(id),
    to_state_id BIGINT REFERENCES states(id),
    from_state_name VARCHAR(255),
    to_state_name VARCHAR(255),
    event_name VARCHAR(100),
    status VARCHAR(50),
    changed_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    comment TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_state_histories_project_id ON state_histories(project_id);
CREATE INDEX IF NOT EXISTS idx_state_histories_created_at ON state_histories(project_id, created_at DESC);

-- Outbox Events (for transactional outbox pattern)
CREATE TABLE IF NOT EXISTS outbox_events (
                                             id BIGSERIAL PRIMARY KEY,
                                             topic VARCHAR(255) NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    aggregate_type VARCHAR(100),
    aggregate_id VARCHAR(100),
    payload JSONB NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 5,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE
                             );

CREATE INDEX IF NOT EXISTS idx_outbox_events_status ON outbox_events(status, created_at)
    WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_outbox_events_aggregate ON outbox_events(aggregate_type, aggregate_id);

-- Form Submissions
CREATE TABLE IF NOT EXISTS form_submissions (
                                                id VARCHAR(36) PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    state_id BIGINT REFERENCES states(id),
    step_name VARCHAR(255),
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    form_type VARCHAR(100),
    data JSONB NOT NULL,
    status VARCHAR(50) DEFAULT 'submitted',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
                                                           );

CREATE INDEX IF NOT EXISTS idx_form_submissions_project_id ON form_submissions(project_id);
CREATE INDEX IF NOT EXISTS idx_form_submissions_state_id ON form_submissions(state_id);
CREATE INDEX IF NOT EXISTS idx_form_submissions_user_id ON form_submissions(user_id);
CREATE INDEX IF NOT EXISTS idx_form_submissions_status ON form_submissions(project_id, status);
