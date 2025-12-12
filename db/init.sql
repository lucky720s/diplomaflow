-- =============================================
-- DiplomaFlow Database Schema
-- Version: 1.0.0
-- =============================================

-- =============================================
-- TABLES
-- =============================================

-- Universities
CREATE TABLE IF NOT EXISTS universities
(
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL UNIQUE,
    short_name VARCHAR(50)  NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
                             );
CREATE INDEX IF NOT EXISTS idx_universities_deleted_at ON universities (deleted_at);

-- Departments
CREATE TABLE IF NOT EXISTS departments
(
    id            BIGSERIAL PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    university_id BIGINT       NOT NULL REFERENCES universities(id) ON DELETE CASCADE,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at    TIMESTAMP WITH TIME ZONE,
                                                                        UNIQUE(name, university_id)
    );
CREATE INDEX IF NOT EXISTS idx_departments_deleted_at ON departments (deleted_at);
CREATE INDEX IF NOT EXISTS idx_departments_university_id ON departments (university_id);

-- Roles
CREATE TABLE IF NOT EXISTS roles
(
    id            BIGSERIAL PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    department_id BIGINT REFERENCES departments(id) ON DELETE SET NULL,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at    TIMESTAMP WITH TIME ZONE,
                                                        UNIQUE(name, department_id)
    );
CREATE INDEX IF NOT EXISTS idx_roles_deleted_at ON roles (deleted_at);
CREATE INDEX IF NOT EXISTS idx_roles_department_id ON roles (department_id);

-- Users
CREATE TABLE IF NOT EXISTS users
(
    id            BIGSERIAL PRIMARY KEY,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password      VARCHAR(255) NOT NULL,
    first_name    VARCHAR(100),
    last_name     VARCHAR(100),
    role          VARCHAR(50)  NOT NULL,
    university_id BIGINT REFERENCES universities(id) ON DELETE SET NULL,
    department_id BIGINT REFERENCES departments(id) ON DELETE SET NULL,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at    TIMESTAMP WITH TIME ZONE
                                                         );
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users (deleted_at);
CREATE INDEX IF NOT EXISTS idx_users_department_id ON users (department_id);
CREATE INDEX IF NOT EXISTS idx_users_university_id ON users (university_id);
CREATE INDEX IF NOT EXISTS idx_users_role ON users (role);

-- Workflows
CREATE TABLE IF NOT EXISTS workflows
(
    id            BIGSERIAL PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    department_id BIGINT       NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    is_active     BOOLEAN                  DEFAULT FALSE,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at    TIMESTAMP WITH TIME ZONE,
                                                                       UNIQUE(name, department_id)
    );
CREATE INDEX IF NOT EXISTS idx_workflows_department_id ON workflows (department_id);
CREATE INDEX IF NOT EXISTS idx_workflows_is_active ON workflows (department_id, is_active) WHERE is_active = true;

-- States
CREATE TABLE IF NOT EXISTS states
(
    id            BIGSERIAL PRIMARY KEY,
    workflow_id   BIGINT       NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    name          VARCHAR(255) NOT NULL,
    description   TEXT,
    type          VARCHAR(50)  NOT NULL,
    config        JSONB                    DEFAULT '{}',
    duration_days INT                      DEFAULT 0,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at    TIMESTAMP WITH TIME ZONE
                                                                     );
CREATE INDEX IF NOT EXISTS idx_states_workflow_id ON states (workflow_id);
CREATE INDEX IF NOT EXISTS idx_states_type ON states (type);

-- Transitions
CREATE TABLE IF NOT EXISTS transitions
(
    id            BIGSERIAL PRIMARY KEY,
    workflow_id   BIGINT       NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    event_name    VARCHAR(255) NOT NULL,
    from_state_id BIGINT       NOT NULL REFERENCES states(id) ON DELETE CASCADE,
    to_state_id   BIGINT       NOT NULL REFERENCES states(id) ON DELETE CASCADE,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(workflow_id, event_name, from_state_id)
    );
CREATE INDEX IF NOT EXISTS idx_transitions_from_event ON transitions (from_state_id, event_name);
CREATE INDEX IF NOT EXISTS idx_transitions_workflow_id ON transitions (workflow_id);

-- Teams (must be before projects due to FK)
CREATE TABLE IF NOT EXISTS teams
(
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    project_id BIGINT UNIQUE,  -- FK added after projects table
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
                             );
CREATE INDEX IF NOT EXISTS idx_teams_deleted_at ON teams (deleted_at);

-- Projects
CREATE TABLE IF NOT EXISTS projects
(
    id                 BIGSERIAL PRIMARY KEY,
    title              VARCHAR(255) NOT NULL,
    description        TEXT,
    student_id         BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    university_id      BIGINT       NOT NULL REFERENCES universities(id) ON DELETE CASCADE,
    department_id      BIGINT       REFERENCES departments(id) ON DELETE SET NULL,
    team_id            BIGINT       REFERENCES teams(id) ON DELETE SET NULL,
    workflow_id        BIGINT       NOT NULL REFERENCES workflows(id) ON DELETE RESTRICT,
    workflow_name      VARCHAR(255),
    current_step_id    VARCHAR(50)  NOT NULL,
    current_state      VARCHAR(255),
    status             VARCHAR(50)              DEFAULT 'active',
    data               JSONB                    DEFAULT '{}',
    deadline_at        TIMESTAMP WITH TIME ZONE,
    deadline_processed BOOLEAN                  DEFAULT FALSE,
    created_at         TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at         TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );
CREATE INDEX IF NOT EXISTS idx_projects_student_id ON projects (student_id);
CREATE INDEX IF NOT EXISTS idx_projects_department_id ON projects (department_id);
CREATE INDEX IF NOT EXISTS idx_projects_workflow_id ON projects (workflow_id);
CREATE INDEX IF NOT EXISTS idx_projects_status ON projects (status);
CREATE INDEX IF NOT EXISTS idx_projects_deadline ON projects (deadline_at) WHERE deadline_processed = false AND status = 'active';

-- Add FK from teams to projects (circular reference)
ALTER TABLE teams DROP CONSTRAINT IF EXISTS fk_teams_project;
ALTER TABLE teams ADD CONSTRAINT fk_teams_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL;

-- State History
CREATE TABLE IF NOT EXISTS state_histories
(
    id         BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    state_name VARCHAR(255),
    status     VARCHAR(50),
    changed_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    comment    TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );
CREATE INDEX IF NOT EXISTS idx_state_histories_project_id ON state_histories (project_id);
CREATE INDEX IF NOT EXISTS idx_state_histories_created_at ON state_histories (project_id, created_at DESC);

-- Team Members
CREATE TABLE IF NOT EXISTS team_members
(
    id         BIGSERIAL PRIMARY KEY,
    team_id    BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(team_id, user_id)
    );
CREATE INDEX IF NOT EXISTS idx_team_members_team_id ON team_members (team_id);
CREATE INDEX IF NOT EXISTS idx_team_members_user_id ON team_members (user_id);

-- Team Invites
CREATE TABLE IF NOT EXISTS team_invites
(
    id         BIGSERIAL PRIMARY KEY,
    team_id    BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    inviter_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status     VARCHAR(50)              DEFAULT 'PENDING',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(team_id, user_id)
    );
CREATE INDEX IF NOT EXISTS idx_team_invites_user_id ON team_invites (user_id);
CREATE INDEX IF NOT EXISTS idx_team_invites_team_id ON team_invites (team_id);
CREATE INDEX IF NOT EXISTS idx_team_invites_status ON team_invites (user_id, status) WHERE status = 'PENDING';

-- Notifications
CREATE TABLE IF NOT EXISTS notifications
(
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title      VARCHAR(255) NOT NULL,
    message    TEXT,
    link       VARCHAR(500),
    type       VARCHAR(50),
    is_read    BOOLEAN                  DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
                                                              );
CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications (user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_unread ON notifications (user_id, is_read, created_at DESC) WHERE is_read = false;

-- Form Submissions
CREATE TABLE IF NOT EXISTS form_submissions
(
    id         VARCHAR(36) PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    step_id    BIGINT NOT NULL,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    data       JSONB  NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
                                                           );
CREATE INDEX IF NOT EXISTS idx_form_submissions_project_id ON form_submissions (project_id);
CREATE INDEX IF NOT EXISTS idx_form_submissions_user_id ON form_submissions (user_id);

-- Outbox Events (for transactional outbox pattern)
CREATE TABLE IF NOT EXISTS outbox_events
(
    id         BIGSERIAL PRIMARY KEY,
    topic      VARCHAR(255) NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    payload    JSONB        NOT NULL,
    status     VARCHAR(50)              DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE
                             );
CREATE INDEX IF NOT EXISTS idx_outbox_events_status ON outbox_events (status, created_at) WHERE status = 'pending';

-- State Actions
CREATE TABLE IF NOT EXISTS state_actions
(
    id       BIGSERIAL PRIMARY KEY,
    state_id BIGINT      NOT NULL REFERENCES states(id) ON DELETE CASCADE,
    type     VARCHAR(50) NOT NULL,
    trigger  VARCHAR(50) NOT NULL,
    config   JSONB       NOT NULL DEFAULT '{}'
    );
CREATE INDEX IF NOT EXISTS idx_state_actions_state_id ON state_actions (state_id);
CREATE INDEX IF NOT EXISTS idx_state_actions_trigger ON state_actions (state_id, trigger);

-- File Metadata
CREATE TABLE IF NOT EXISTS file_metadata
(
    id         VARCHAR(36) PRIMARY KEY,
    user_id    BIGINT REFERENCES users(id) ON DELETE SET NULL,
    project_id BIGINT REFERENCES projects(id) ON DELETE SET NULL,
    file_name  VARCHAR(255),
    file_type  VARCHAR(50),
    size       BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );
CREATE INDEX IF NOT EXISTS idx_file_metadata_user_id ON file_metadata (user_id);
CREATE INDEX IF NOT EXISTS idx_file_metadata_project_id ON file_metadata (project_id);

-- Refresh Tokens (for auth)
CREATE TABLE IF NOT EXISTS refresh_tokens
(
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      VARCHAR(255) NOT NULL UNIQUE,
    user_agent VARCHAR(500),
    client_ip  VARCHAR(45),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked    BOOLEAN                  DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON refresh_tokens (token) WHERE revoked = false;
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires ON refresh_tokens (expires_at) WHERE revoked = false;

-- =============================================
-- SEED DATA
-- =============================================

-- Universities
INSERT INTO universities (name, short_name)
VALUES
    ('International Information Technology University', 'IITU'),
    ('Astana IT University', 'AITU')
    ON CONFLICT (name) DO NOTHING;

-- Departments for IITU (university_id = 1)
INSERT INTO departments (name, university_id)
VALUES
    ('Computer Science', 1),
    ('Software Engineering', 1)
    ON CONFLICT (name, university_id) DO NOTHING;

-- Departments for AITU (university_id = 2)
INSERT INTO departments (name, university_id)
VALUES
    ('Information Systems', 2),
    ('Computer Engineering', 2),
    ('Radio Engineering', 2)
    ON CONFLICT (name, university_id) DO NOTHING;

-- Roles
INSERT INTO roles (name, department_id)
VALUES
    ('Senior Lecturer', 1),
    ('Professor', 1),
    ('Associate Professor', 1)
    ON CONFLICT (name, department_id) DO NOTHING;

-- Users (Password for all: '12345678', bcrypt hash with cost 14)
INSERT INTO users (email, password, first_name, last_name, role, university_id, department_id)
VALUES
    ('student@example.com',
     '$2a$14$ajq8Q7fbtFRQvXikDrten.rXMkmy.mQdIT.Z5JtM9.g/q.wj.uAm',
     'Ivan', 'Ivanov', 'student', 1, 1),
    ('student2@example.com',
     '$2a$14$ajq8Q7fbtFRQvXikDrten.rXMkmy.mQdIT.Z5JtM9.g/q.wj.uAm',
     'Petr', 'Sidorov', 'student', 1, 1),
    ('student3@example.com',
     '$2a$14$ajq8Q7fbtFRQvXikDrten.rXMkmy.mQdIT.Z5JtM9.g/q.wj.uAm',
     'Maria', 'Petrova', 'student', 1, 1),
    ('teacher@example.com',
     '$2a$14$ajq8Q7fbtFRQvXikDrten.rXMkmy.mQdIT.Z5JtM9.g/q.wj.uAm',
     'Petr', 'Petrov', 'teacher', 1, 1),
    ('aitu_student@example.com',
     '$2a$14$ajq8Q7fbtFRQvXikDrten.rXMkmy.mQdIT.Z5JtM9.g/q.wj.uAm',
     'Alibek', 'Alibekov', 'student', 2, 3),
    ('admin@example.com',
     '$2a$14$ajq8Q7fbtFRQvXikDrten.rXMkmy.mQdIT.Z5JtM9.g/q.wj.uAm',
     'Admin', 'System', 'admin', 1, NULL)
    ON CONFLICT (email) DO NOTHING;

-- Workflow
INSERT INTO workflows (name, department_id, is_active)
VALUES ('Diploma Process 2025', 1, true)
    ON CONFLICT (name, department_id) DO NOTHING;

-- =============================================
-- STATES
-- ВАЖНО: Все states должны быть созданы ДО transitions!
-- =============================================
INSERT INTO states (workflow_id, name, description, type, config, duration_days)
VALUES
    -- id = 1: Team Formation
    (1, 'Team Formation', 'Students form teams and select teammates',
     'TEAM_FORMATION',
     '{"team_config": {"min_size": 1, "max_size": 3, "allow_solo": true}}',
     7),

    -- id = 2: Topic Selection
    (1, 'Topic Selection', 'Select supervisor and propose diploma topic',
     'SUPERVISOR_SELECTION',
     '{"allowed_roles": ["teacher", "professor"]}',
     5),

    -- id = 3: Task Upload
    (1, 'Task Upload', 'Upload signed task document for diploma',
     'DOCUMENT_UPLOAD',
     '{"file_requirements": {"allowed_extensions": [".pdf", ".doc", ".docx"], "max_size_bytes": 10485760, "max_files": 1}}',
     3),

    -- id = 4: Final Approval
    (1, 'Final Approval', 'Department head approves the diploma project',
     'APPROVAL',
     '{"allowed_roles": ["admin", "head_of_department"]}',
     3),

    -- id = 5: Completed
    (1, 'Completed', 'Diploma process successfully completed',
     'APPROVAL',
     '{}',
     0)
    ON CONFLICT DO NOTHING;

-- =============================================
-- TRANSITIONS
-- ВАЖНО: Должны создаваться ПОСЛЕ всех states!
-- =============================================
INSERT INTO transitions (workflow_id, event_name, from_state_id, to_state_id)
VALUES
    -- Stage 1 → Stage 2 (Team formed, move to topic selection)
    (1, 'TEAM_FORMED', 1, 2),

    -- Stage 2 → Stage 3 (Topic approved by supervisor)
    (1, 'TOPIC_APPROVED', 2, 3),

    -- Stage 3 → Stage 4 (Task document uploaded)
    (1, 'TASK_UPLOADED', 3, 4),

    -- Stage 4 → Stage 5 (Final approval - completed)
    (1, 'APPROVE', 4, 5),

    -- Rejection paths (return to previous stage)
    (1, 'REJECT', 4, 3),  -- Rejected at approval → back to upload
    (1, 'REJECT', 3, 2)   -- Rejected at upload → back to topic selection
    ON CONFLICT (workflow_id, event_name, from_state_id) DO NOTHING;

-- =============================================
-- STATE ACTIONS (notifications/webhooks)
-- =============================================
INSERT INTO state_actions (state_id, type, trigger, config)
VALUES
    -- Notify student when entering Team Formation
    (1, 'SEND_NOTIFICATION', 'ON_ENTER',
     '{"title": "Начните формирование команды", "message": "Вы можете пригласить до 2 участников в команду", "link": "/teams"}'),

    -- Notify when topic needs to be selected
    (2, 'SEND_NOTIFICATION', 'ON_ENTER',
     '{"title": "Выберите тему и руководителя", "message": "Свяжитесь с преподавателем для выбора темы диплома", "link": "/projects/{project_id}"}'),

    -- Notify about deadline approaching (24h before)
    (3, 'SEND_NOTIFICATION', 'ON_DEADLINE',
     '{"title": "Срок загрузки истекает!", "message": "Осталось менее 24 часов для загрузки документа", "urgency": "high"}'),

    -- Notify when awaiting approval
    (4, 'SEND_NOTIFICATION', 'ON_ENTER',
     '{"title": "Ожидает одобрения", "message": "Ваш проект отправлен на финальное одобрение"}'),

    -- Notify on completion
    (5, 'SEND_NOTIFICATION', 'ON_ENTER',
     '{"title": "Поздравляем! 🎉", "message": "Ваш дипломный проект одобрен"}')
    ON CONFLICT DO NOTHING;

-- =============================================
-- VERIFICATION QUERIES (for debugging)
-- =============================================

-- Verify states count
DO $$
DECLARE
state_count INTEGER;
BEGIN
SELECT COUNT(*) INTO state_count FROM states WHERE workflow_id = 1;
IF state_count < 5 THEN
        RAISE WARNING 'Expected 5 states, found %', state_count;
ELSE
        RAISE NOTICE 'States OK: % states created', state_count;
END IF;
END $$;

-- Verify transitions count
DO $$
DECLARE
trans_count INTEGER;
BEGIN
SELECT COUNT(*) INTO trans_count FROM transitions WHERE workflow_id = 1;
IF trans_count < 6 THEN
        RAISE WARNING 'Expected 6 transitions, found %', trans_count;
ELSE
        RAISE NOTICE 'Transitions OK: % transitions created', trans_count;
END IF;
END $$;

-- Verify all transitions reference valid states
DO $$
DECLARE
invalid_count INTEGER;
BEGIN
SELECT COUNT(*) INTO invalid_count
FROM transitions t
WHERE NOT EXISTS (SELECT 1 FROM states s WHERE s.id = t.from_state_id)
   OR NOT EXISTS (SELECT 1 FROM states s WHERE s.id = t.to_state_id);

IF invalid_count > 0 THEN
        RAISE EXCEPTION 'Found % transitions with invalid state references!', invalid_count;
ELSE
        RAISE NOTICE 'All transitions reference valid states';
END IF;
END $$;
