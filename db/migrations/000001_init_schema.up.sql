BEGIN;

-- Extensions (for bcrypt hashes in seeds)
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- =========================
-- 0) Core reference tables
-- =========================

CREATE TABLE IF NOT EXISTS universities (
                                            id BIGSERIAL PRIMARY KEY,
                                            name VARCHAR(255) NOT NULL UNIQUE,
    short_name VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
    );
CREATE INDEX IF NOT EXISTS idx_universities_deleted_at ON universities(deleted_at);

CREATE TABLE IF NOT EXISTS departments (
                                           id BIGSERIAL PRIMARY KEY,
                                           name VARCHAR(255) NOT NULL,
    university_id BIGINT NOT NULL REFERENCES universities(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(name, university_id)
    );
CREATE INDEX IF NOT EXISTS idx_departments_university_id ON departments(university_id);
CREATE INDEX IF NOT EXISTS idx_departments_deleted_at ON departments(deleted_at);

-- Directory of additional roles/positions/capabilities (used by role_service, workflow configs, admin)
CREATE TABLE IF NOT EXISTS roles (
                                     id BIGSERIAL PRIMARY KEY,
                                     name VARCHAR(255) NOT NULL,
    department_id BIGINT REFERENCES departments(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(name, department_id)
    );
CREATE INDEX IF NOT EXISTS idx_roles_department_id ON roles(department_id);
CREATE INDEX IF NOT EXISTS idx_roles_deleted_at ON roles(deleted_at);

-- =========================
-- 1) Users & Auth
-- =========================

-- Base role for JWT/gateway RBAC: student|teacher|admin (single string, as in auth.proto)
CREATE TABLE IF NOT EXISTS users (
                                     id BIGSERIAL PRIMARY KEY,
                                     email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role VARCHAR(50) NOT NULL,
    university_id BIGINT REFERENCES universities(id) ON DELETE SET NULL,
    department_id BIGINT REFERENCES departments(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
    );
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
CREATE INDEX IF NOT EXISTS idx_users_department_id ON users(department_id);
CREATE INDEX IF NOT EXISTS idx_users_university_id ON users(university_id);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Dynamic/additional roles assignment per department/workflow scope
CREATE TABLE IF NOT EXISTS user_role_assignments (
                                                     id BIGSERIAL PRIMARY KEY,
                                                     user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    department_id BIGINT REFERENCES departments(id) ON DELETE SET NULL,
    workflow_id BIGINT,
    assigned_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    comment TEXT
    );
CREATE UNIQUE INDEX IF NOT EXISTS ux_user_role_scope
    ON user_role_assignments(user_id, role_id, COALESCE(department_id, 0), COALESCE(workflow_id, 0))
    WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_role_user ON user_role_assignments(user_id);
CREATE INDEX IF NOT EXISTS idx_user_role_role ON user_role_assignments(role_id);
CREATE INDEX IF NOT EXISTS idx_user_role_department ON user_role_assignments(department_id);

CREATE TABLE IF NOT EXISTS refresh_tokens (
                                              id BIGSERIAL PRIMARY KEY,
                                              user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    user_agent VARCHAR(500),
    client_ip VARCHAR(45),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON refresh_tokens(token) WHERE revoked = false;

-- =========================
-- 2) Teams (team exists BEFORE project)
-- =========================

CREATE TABLE IF NOT EXISTS teams (
                                     id BIGSERIAL PRIMARY KEY,
                                     name VARCHAR(255) NOT NULL,

    university_id BIGINT NOT NULL REFERENCES universities(id) ON DELETE RESTRICT,
    department_id BIGINT NOT NULL REFERENCES departments(id) ON DELETE RESTRICT,

    invite_code VARCHAR(6) NOT NULL,
    composition_locked BOOLEAN NOT NULL DEFAULT FALSE,
    composition_locked_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
    );
CREATE INDEX IF NOT EXISTS idx_teams_deleted_at ON teams(deleted_at);
CREATE INDEX IF NOT EXISTS idx_teams_department_id ON teams(department_id);
CREATE UNIQUE INDEX IF NOT EXISTS ux_teams_university_invite_code ON teams(university_id, invite_code);

CREATE TABLE IF NOT EXISTS team_members (
                                            id BIGSERIAL PRIMARY KEY,
                                            team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'member', -- leader, member
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(team_id, user_id)
    );
CREATE INDEX IF NOT EXISTS idx_team_members_team_id ON team_members(team_id);
CREATE INDEX IF NOT EXISTS idx_team_members_user_id ON team_members(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS ux_team_leader ON team_members(team_id) WHERE role = 'leader';

CREATE TABLE IF NOT EXISTS team_invites (
                                            id BIGSERIAL PRIMARY KEY,
                                            team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    inviter_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING', -- PENDING/ACCEPTED/REJECTED/CANCELLED
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(team_id, user_id)
    );
CREATE INDEX IF NOT EXISTS idx_team_invites_user_id ON team_invites(user_id);
CREATE INDEX IF NOT EXISTS idx_team_invites_pending ON team_invites(user_id, status) WHERE status = 'PENDING';
CREATE INDEX IF NOT EXISTS idx_team_invites_expires_at ON team_invites(expires_at);

-- =========================
-- 3) Notifications & Files
-- =========================

CREATE TABLE IF NOT EXISTS notifications (
                                             id BIGSERIAL PRIMARY KEY,
                                             user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    message TEXT,
    link VARCHAR(500),
    type VARCHAR(50),
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
    );
CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_unread ON notifications(user_id, is_read, created_at DESC) WHERE is_read = false;

CREATE TABLE IF NOT EXISTS file_metadata (
                                             id VARCHAR(36) PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    project_id BIGINT,
    file_name VARCHAR(255),
    file_type VARCHAR(50),
    size BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );
CREATE INDEX IF NOT EXISTS idx_file_metadata_user_id ON file_metadata(user_id);

-- =========================
-- 4) Workflow directory
-- =========================

CREATE TABLE IF NOT EXISTS workflows (
                                         id BIGSERIAL PRIMARY KEY,
                                         name VARCHAR(255) NOT NULL,
    description TEXT,
    department_id BIGINT NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    version INT NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    is_template BOOLEAN NOT NULL DEFAULT FALSE,
    parent_id BIGINT REFERENCES workflows(id) ON DELETE SET NULL,
    settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(name, department_id, version)
    );
CREATE INDEX IF NOT EXISTS idx_workflows_department_id ON workflows(department_id);
CREATE INDEX IF NOT EXISTS idx_workflows_is_active ON workflows(department_id, is_active) WHERE is_active = true;

CREATE TABLE IF NOT EXISTS states (
                                      id BIGSERIAL PRIMARY KEY,
                                      workflow_id BIGINT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    order_index INT NOT NULL DEFAULT 0,
    type VARCHAR(50) NOT NULL, -- enum name from workflow.v1.StateType

    is_initial BOOLEAN NOT NULL DEFAULT FALSE,
    is_final BOOLEAN NOT NULL DEFAULT FALSE,
    is_optional BOOLEAN NOT NULL DEFAULT FALSE,

    config JSONB NOT NULL DEFAULT '{}'::jsonb,

    duration_days INT NOT NULL DEFAULT 0,
    duration_mode VARCHAR(20) NOT NULL DEFAULT 'relative',
    fixed_deadline TIMESTAMPTZ,

    color VARCHAR(20),
    icon VARCHAR(50),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
    );
CREATE INDEX IF NOT EXISTS idx_states_workflow_id ON states(workflow_id);
CREATE INDEX IF NOT EXISTS idx_states_order ON states(workflow_id, order_index);

CREATE TABLE IF NOT EXISTS transitions (
                                           id BIGSERIAL PRIMARY KEY,
                                           workflow_id BIGINT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    event_name VARCHAR(100) NOT NULL,
    display_name VARCHAR(255),
    from_state_id BIGINT NOT NULL REFERENCES states(id) ON DELETE CASCADE,
    to_state_id BIGINT NOT NULL REFERENCES states(id) ON DELETE CASCADE,
    conditions JSONB NOT NULL DEFAULT '[]'::jsonb,
    button_label VARCHAR(100),
    button_color VARCHAR(20) NOT NULL DEFAULT 'primary',
    confirm_text TEXT,
    priority INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(workflow_id, event_name, from_state_id)
    );
CREATE INDEX IF NOT EXISTS idx_transitions_from_state ON transitions(from_state_id);

CREATE TABLE IF NOT EXISTS state_actions (
                                             id BIGSERIAL PRIMARY KEY,
                                             state_id BIGINT NOT NULL REFERENCES states(id) ON DELETE CASCADE,
    name VARCHAR(100),
    type VARCHAR(50) NOT NULL,
    trigger VARCHAR(50) NOT NULL,
    order_index INT NOT NULL DEFAULT 0,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    is_optional BOOLEAN NOT NULL DEFAULT FALSE,
    conditions JSONB NOT NULL DEFAULT '[]'::jsonb,
    max_retries INT NOT NULL DEFAULT 3,
    retry_delay_seconds INT NOT NULL DEFAULT 60,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );
CREATE INDEX IF NOT EXISTS idx_state_actions_state_id ON state_actions(state_id);

CREATE TABLE IF NOT EXISTS state_conditions (
                                                id BIGSERIAL PRIMARY KEY,
                                                state_id BIGINT NOT NULL REFERENCES states(id) ON DELETE CASCADE,
    name VARCHAR(100),
    type VARCHAR(50) NOT NULL,
    expression TEXT,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS workflow_templates (
                                                  id BIGSERIAL PRIMARY KEY,
                                                  name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    category VARCHAR(100),
    template_data JSONB NOT NULL,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

-- Department workflow configs (used by GetDepartmentWorkflowConfig/GetTeamConfiguration)
CREATE TABLE IF NOT EXISTS department_workflow_configs (
                                                           id BIGSERIAL PRIMARY KEY,
                                                           department_id BIGINT NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    workflow_id BIGINT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    academic_year VARCHAR(20) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    config_overrides JSONB NOT NULL DEFAULT '{}'::jsonb,
    team_settings JSONB NOT NULL DEFAULT '{
    "allow_solo": true,
    "min_size": 1,
    "max_size": 4,
    "require_same_group": false,
    "invite_expire_days": 3
  }'::jsonb,
    deadline_overrides JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(department_id, academic_year)
    );
CREATE INDEX IF NOT EXISTS idx_dept_wf_config_active ON department_workflow_configs(department_id, is_active) WHERE is_active = true;

CREATE TABLE IF NOT EXISTS department_custom_steps (
                                                       id BIGSERIAL PRIMARY KEY,
                                                       department_config_id BIGINT NOT NULL REFERENCES department_workflow_configs(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    step_type VARCHAR(50) NOT NULL,
    insert_after_state_id BIGINT REFERENCES states(id) ON DELETE SET NULL,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_required BOOLEAN NOT NULL DEFAULT TRUE,
    duration_days INT NOT NULL DEFAULT 7,
    order_index INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS action_registry (
                                               id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50),
    config_schema JSONB NOT NULL,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

-- =========================
-- 5) Projects runtime
-- =========================

CREATE TABLE IF NOT EXISTS projects (
                                        id BIGSERIAL PRIMARY KEY,
                                        title VARCHAR(255) NOT NULL,
    description TEXT,

    student_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    university_id BIGINT NOT NULL REFERENCES universities(id) ON DELETE CASCADE,
    department_id BIGINT REFERENCES departments(id) ON DELETE SET NULL,
    team_id BIGINT REFERENCES teams(id) ON DELETE SET NULL,

    workflow_id BIGINT NOT NULL REFERENCES workflows(id) ON DELETE RESTRICT,
    workflow_version INT NOT NULL DEFAULT 1,
    workflow_name VARCHAR(255),

    current_state_id BIGINT REFERENCES states(id) ON DELETE SET NULL,
    current_state_name VARCHAR(255),

    status VARCHAR(50) NOT NULL DEFAULT 'active',
    data JSONB NOT NULL DEFAULT '{}'::jsonb,

    deadline_at TIMESTAMPTZ,
    deadline_processed BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );
CREATE INDEX IF NOT EXISTS idx_projects_student_id ON projects(student_id);
CREATE INDEX IF NOT EXISTS idx_projects_team_id ON projects(team_id);
CREATE UNIQUE INDEX IF NOT EXISTS ux_projects_team_id ON projects(team_id) WHERE team_id IS NOT NULL;

ALTER TABLE file_metadata
    ADD CONSTRAINT fk_file_metadata_project
        FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_file_metadata_project_id ON file_metadata(project_id);

CREATE TABLE IF NOT EXISTS state_histories (
                                               id BIGSERIAL PRIMARY KEY,
                                               project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    from_state_id BIGINT REFERENCES states(id) ON DELETE SET NULL,
    to_state_id BIGINT REFERENCES states(id) ON DELETE SET NULL,
    from_state_name VARCHAR(255),
    to_state_name VARCHAR(255),
    event_name VARCHAR(100),
    status VARCHAR(50),
    changed_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    comment TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );
CREATE INDEX IF NOT EXISTS idx_state_histories_project_id ON state_histories(project_id);

CREATE TABLE IF NOT EXISTS outbox_events (
                                             id BIGSERIAL PRIMARY KEY,
                                             topic VARCHAR(255) NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    aggregate_type VARCHAR(100),
    aggregate_id VARCHAR(100),
    payload JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 5,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
    );
CREATE INDEX IF NOT EXISTS idx_outbox_events_pending ON outbox_events(status, created_at) WHERE status = 'pending';

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
    status VARCHAR(30) NOT NULL DEFAULT 'running',
    attempts INT NOT NULL DEFAULT 1,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    succeeded_at TIMESTAMPTZ
    );

-- Form submissions (form_service)
CREATE TABLE IF NOT EXISTS form_submissions (
                                                id VARCHAR(36) PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    state_id BIGINT REFERENCES states(id) ON DELETE SET NULL,
    step_name VARCHAR(255),
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    form_type VARCHAR(100),
    data JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'submitted',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
    );

-- =========================
-- 6) Admin subsystem
-- =========================

CREATE TABLE IF NOT EXISTS admin_grades (
                                            id BIGSERIAL PRIMARY KEY,
                                            project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    state_id BIGINT REFERENCES states(id) ON DELETE SET NULL,
    team_id BIGINT REFERENCES teams(id) ON DELETE SET NULL,
    grade INT NOT NULL CHECK (grade >= 0 AND grade <= 100),
    comment TEXT,
    graded_by BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(project_id, state_id)
    );

CREATE TABLE IF NOT EXISTS admin_grade_history (
                                                   id BIGSERIAL PRIMARY KEY,
                                                   grade_id BIGINT NOT NULL REFERENCES admin_grades(id) ON DELETE CASCADE,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    state_id BIGINT REFERENCES states(id) ON DELETE SET NULL,
    old_grade INT,
    new_grade INT NOT NULL,
    changed_by BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS admin_submissions (
                                                 id VARCHAR(36) PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    team_id BIGINT REFERENCES teams(id) ON DELETE SET NULL,
    state_id BIGINT REFERENCES states(id) ON DELETE SET NULL,
    submitted_by BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    data JSONB NOT NULL DEFAULT '{}'::jsonb,
    files JSONB NOT NULL DEFAULT '[]'::jsonb,
    reviewer_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    review_comment TEXT,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
    );

CREATE TABLE IF NOT EXISTS admin_submission_reviews (
                                                        id BIGSERIAL PRIMARY KEY,
                                                        submission_id VARCHAR(36) NOT NULL REFERENCES admin_submissions(id) ON DELETE CASCADE,
    reviewer_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action VARCHAR(30) NOT NULL,
    comment TEXT,
    grade INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS admin_supervisor_assignments (
                                                            id BIGSERIAL PRIMARY KEY,
                                                            team_id BIGINT NOT NULL UNIQUE REFERENCES teams(id) ON DELETE CASCADE,
    supervisor_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    assigned_by BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS admin_activities (
                                                id BIGSERIAL PRIMARY KEY,
                                                activity_type VARCHAR(50) NOT NULL,
    description TEXT,
    actor_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    target_id BIGINT,
    target_type VARCHAR(50),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS admin_topic_registrations (
                                                         id VARCHAR(36) PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    proposed_topic VARCHAR(500) NOT NULL,
    topic_description TEXT,
    supervisor_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    submitted_by BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    rejection_reason TEXT,
    comment TEXT,
    reviewer_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
    );

CREATE TABLE IF NOT EXISTS admin_topic_registration_reviews (
                                                                id BIGSERIAL PRIMARY KEY,
                                                                registration_id VARCHAR(36) NOT NULL REFERENCES admin_topic_registrations(id) ON DELETE CASCADE,
    reviewer_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action VARCHAR(30) NOT NULL,
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

-- Supervisor requests: team-first supported (project_id nullable)
CREATE TABLE IF NOT EXISTS admin_supervisor_requests (
                                                         id VARCHAR(36) PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    project_id BIGINT REFERENCES projects(id) ON DELETE SET NULL,
    supervisor_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    requested_by BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    message TEXT,
    proposed_topic VARCHAR(500),
    reject_reason TEXT,
    responded_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS admin_supervisor_request_history (
                                                                id BIGSERIAL PRIMARY KEY,
                                                                request_id VARCHAR(36) NOT NULL REFERENCES admin_supervisor_requests(id) ON DELETE CASCADE,
    action VARCHAR(30) NOT NULL,
    actor_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

-- Pre-defense
CREATE TABLE IF NOT EXISTS admin_pre_defense_submissions (
                                                             id VARCHAR(36) PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    supervisor_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    submitted_by BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    scheduled_date TIMESTAMPTZ,
    scheduled_time VARCHAR(10),
    location VARCHAR(255),
    meeting_link VARCHAR(500),
    duration_minutes INT NOT NULL DEFAULT 30,
    grade INT CHECK (grade >= 0 AND grade <= 100),
    grade_comment TEXT,
    graded_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    graded_at TIMESTAMPTZ,
    result VARCHAR(30),
    result_comment TEXT,
    recommendations TEXT[],
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS admin_pre_defense_commission (
                                                            id BIGSERIAL PRIMARY KEY,
                                                            submission_id VARCHAR(36) NOT NULL REFERENCES admin_pre_defense_submissions(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    role VARCHAR(30) NOT NULL,
    is_present BOOLEAN NOT NULL DEFAULT FALSE,
    individual_grade INT,
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS admin_pre_defense_documents (
                                                           id VARCHAR(36) PRIMARY KEY,
    submission_id VARCHAR(36) NOT NULL REFERENCES admin_pre_defense_submissions(id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL,
    file_type VARCHAR(50),
    display_name VARCHAR(255),
    size BIGINT,
    download_url VARCHAR(500),
    uploaded_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS admin_pre_defense_history (
                                                         id BIGSERIAL PRIMARY KEY,
                                                         submission_id VARCHAR(36) NOT NULL REFERENCES admin_pre_defense_submissions(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    actor_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    old_value TEXT,
    new_value TEXT,
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

-- =========================
-- 7) Task management
-- =========================

CREATE TABLE IF NOT EXISTS task_boards (
                                           id BIGSERIAL PRIMARY KEY,
                                           team_id BIGINT NOT NULL UNIQUE REFERENCES teams(id) ON DELETE CASCADE,
    project_id BIGINT REFERENCES projects(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL DEFAULT 'Доска задач',
    description TEXT,
    settings JSONB NOT NULL DEFAULT '{
    "default_column":"todo",
    "allow_custom_columns":false,
    "show_completed":true,
    "labels":["backend","frontend","docs","research","urgent"]
  }'::jsonb,
    created_by BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
    );

CREATE TABLE IF NOT EXISTS task_columns (
                                            id BIGSERIAL PRIMARY KEY,
                                            board_id BIGINT NOT NULL REFERENCES task_boards(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) NOT NULL,
    description TEXT,
    color VARCHAR(20) DEFAULT '#6B7280',
    icon VARCHAR(50),
    order_index INT NOT NULL DEFAULT 0,
    wip_limit INT NOT NULL DEFAULT 0,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    is_done_column BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(board_id, slug)
    );

CREATE TABLE IF NOT EXISTS tasks (
                                     id BIGSERIAL PRIMARY KEY,
                                     board_id BIGINT NOT NULL REFERENCES task_boards(id) ON DELETE CASCADE,
    column_id BIGINT NOT NULL REFERENCES task_columns(id) ON DELETE RESTRICT,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    status VARCHAR(30) NOT NULL DEFAULT 'todo',
    priority VARCHAR(20) NOT NULL DEFAULT 'medium',
    assignee_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_by BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    due_date DATE,
    due_time TIME,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    estimated_minutes INT NOT NULL DEFAULT 0,
    actual_minutes INT NOT NULL DEFAULT 0,
    position INT NOT NULL DEFAULT 0,
    workflow_step_id BIGINT REFERENCES states(id) ON DELETE SET NULL,
    labels JSONB NOT NULL DEFAULT '[]'::jsonb,
    custom_fields JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
    );

CREATE TABLE IF NOT EXISTS task_comments (
                                             id BIGSERIAL PRIMARY KEY,
                                             task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    author_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    content TEXT NOT NULL,
    mentions JSONB NOT NULL DEFAULT '[]'::jsonb,
    edited_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
    );

CREATE TABLE IF NOT EXISTS task_attachments (
                                                id BIGSERIAL PRIMARY KEY,
                                                task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    file_id VARCHAR(36) REFERENCES file_metadata(id) ON DELETE SET NULL,
    file_name VARCHAR(255) NOT NULL,
    file_type VARCHAR(100),
    file_size BIGINT NOT NULL DEFAULT 0,
    uploaded_by BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS task_activity_log (
                                                 id BIGSERIAL PRIMARY KEY,
                                                 task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    actor_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action VARCHAR(50) NOT NULL,
    field_name VARCHAR(100),
    old_value TEXT,
    new_value TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS task_watchers (
                                             id BIGSERIAL PRIMARY KEY,
                                             task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(task_id, user_id)
    );

COMMIT;
