CREATE TABLE IF NOT EXISTS universities
(
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL UNIQUE,
    short_name VARCHAR(50)  NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
                             );
CREATE INDEX idx_universities_deleted_at ON universities (deleted_at);

CREATE TABLE IF NOT EXISTS departments
(
    id            BIGSERIAL PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    university_id BIGINT       NOT NULL,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at    TIMESTAMP WITH TIME ZONE
                                );
CREATE INDEX idx_departments_deleted_at ON departments (deleted_at);
CREATE INDEX idx_departments_university_id ON departments (university_id);

CREATE TABLE IF NOT EXISTS roles
(
    id            BIGSERIAL PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    department_id BIGINT,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at    TIMESTAMP WITH TIME ZONE
                                );
CREATE INDEX idx_roles_deleted_at ON roles (deleted_at);

CREATE TABLE IF NOT EXISTS users
(
    id            BIGSERIAL PRIMARY KEY,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password      VARCHAR(255) NOT NULL,
    first_name    VARCHAR(100),
    last_name     VARCHAR(100),
    role          VARCHAR(50)  NOT NULL,
    university_id BIGINT,
    department_id BIGINT,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at    TIMESTAMP WITH TIME ZONE
                                );
CREATE INDEX idx_users_deleted_at ON users (deleted_at);
CREATE INDEX idx_users_department_id ON users (department_id);

CREATE TABLE IF NOT EXISTS workflows
(
    id            BIGSERIAL PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    department_id BIGINT       NOT NULL,
    is_active     BOOLEAN                  DEFAULT FALSE,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at    TIMESTAMP WITH TIME ZONE
                                );
CREATE INDEX idx_workflows_department_id ON workflows (department_id);

CREATE TABLE IF NOT EXISTS states
(
    id            BIGSERIAL PRIMARY KEY,
    workflow_id   BIGINT       NOT NULL,
    name          VARCHAR(255) NOT NULL,
    description   TEXT,
    type          INT          NOT NULL,
    config        JSONB                    DEFAULT '{}',
    duration_days INT                      DEFAULT 0,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at    TIMESTAMP WITH TIME ZONE
                                );
CREATE INDEX idx_states_workflow_id ON states (workflow_id);

CREATE TABLE IF NOT EXISTS transitions
(
    id            BIGSERIAL PRIMARY KEY,
    workflow_id   BIGINT       NOT NULL,
    event_name    VARCHAR(255) NOT NULL,
    from_state_id BIGINT       NOT NULL,
    to_state_id   BIGINT       NOT NULL,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );


INSERT INTO universities (name, short_name)
VALUES ('International Information Technology University', 'IITU'),('Astana IT University', 'AITU');

INSERT INTO departments (name, university_id)
VALUES ('Computer Science', 1),
       ('Software Engineering', 1);

INSERT INTO departments (name, university_id)
VALUES ('Information Systems', 2),
       ('Computer Engineering', 2),
       ('Radio Engineering', 2);

INSERT INTO roles (name, department_id)
VALUES ('Senior Lecturer', 1),
       ('Professor', 1);

-- Пароль для всех: '12345678' (хэш bcrypt cost 14)
INSERT INTO users (email, password, first_name, last_name, role, university_id, department_id)
VALUES ('student@example.com', '$2a$14$ajq8Q7fbtFRQvXikDrten.rXMkmy.mQdIT.Z5JtM9.g/q.wj.uAm', 'Ivan', 'Ivanov',
        'student', 1, 1), -- AITU, CS
       ('teacher@example.com', '$2a$14$ajq8Q7fbtFRQvXikDrten.rXMkmy.mQdIT.Z5JtM9.g/q.wj.uAm', 'Petr', 'Petrov',
        'teacher', 1, 1), -- AITU, CS
       ('muit_student@example.com', '$2a$14$ajq8Q7fbtFRQvXikDrten.rXMkmy.mQdIT.Z5JtM9.g/q.wj.uAm', 'Alibek', 'Alibekov',
        'student', 2, 3), -- MUIT, IS
       ('admin@example.com', '$2a$14$ajq8Q7fbtFRQvXikDrten.rXMkmy.mQdIT.Z5JtM9.g/q.wj.uAm', 'Admin', 'System', 'admin',
        1, NULL);

INSERT INTO workflows (name, department_id, is_active)
VALUES ('Diploma Process 2025', 1, true);

INSERT INTO states (workflow_id, name, description, type, config, duration_days)
VALUES (1, 'Team Formation', 'Students form teams', 1,
        '{"team_config": {"min_size": 1, "max_size": 3, "allow_solo": true}}', 7);

INSERT INTO states (workflow_id, name, description, type, config, duration_days)
VALUES (1, 'Topic Selection', 'Select supervisor and topic', 2,
        '{}', 5);

INSERT INTO states (workflow_id, name, description, type, config, duration_days)
VALUES (1, 'Task Upload', 'Upload signed task document', 3,
        '{"file_requirements": {"allowed_extensions": [".pdf"], "max_size_bytes": 10485760}}', 3);

INSERT INTO states (workflow_id, name, description, type, config, duration_days)
VALUES (1, 'Final Approval', 'Department approval', 6,
        '{}', 3);

INSERT INTO transitions (workflow_id, event_name, from_state_id, to_state_id)
VALUES (1, 'TEAM_FORMED', 1, 2);

INSERT INTO transitions (workflow_id, event_name, from_state_id, to_state_id)
VALUES (1, 'TOPIC_APPROVED', 2, 3);

INSERT INTO transitions (workflow_id, event_name, from_state_id, to_state_id)
VALUES (1, 'TASK_UPLOADED', 3, 4);
