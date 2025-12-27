-- =============================================
-- Migration: 000004_admin
-- Description: Admin panel tables
-- =============================================

-- Grades
CREATE TABLE IF NOT EXISTS admin_grades (
                                            id BIGSERIAL PRIMARY KEY,
                                            project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    state_id BIGINT REFERENCES states(id),
    team_id BIGINT REFERENCES teams(id),
    grade INT NOT NULL CHECK (grade >= 0 AND grade <= 100),
    letter_grade VARCHAR(2),
    comment TEXT,
    graded_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
                                                                                                   UNIQUE(project_id, state_id)
    );

CREATE INDEX IF NOT EXISTS idx_admin_grades_project ON admin_grades(project_id);
CREATE INDEX IF NOT EXISTS idx_admin_grades_state ON admin_grades(state_id);

-- Grade History
CREATE TABLE IF NOT EXISTS admin_grade_history (
                                                   id BIGSERIAL PRIMARY KEY,
                                                   grade_id BIGINT NOT NULL REFERENCES admin_grades(id) ON DELETE CASCADE,
    project_id BIGINT NOT NULL REFERENCES projects(id),
    state_id BIGINT REFERENCES states(id),
    old_grade INT,
    new_grade INT NOT NULL,
    changed_by BIGINT NOT NULL REFERENCES users(id),
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_admin_grade_history_grade ON admin_grade_history(grade_id);
CREATE INDEX IF NOT EXISTS idx_admin_grade_history_project ON admin_grade_history(project_id);

-- Submissions (для проверки)
CREATE TABLE IF NOT EXISTS admin_submissions (
                                                 id VARCHAR(36) PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    team_id BIGINT REFERENCES teams(id),
    state_id BIGINT REFERENCES states(id),
    submitted_by BIGINT NOT NULL REFERENCES users(id),
    status VARCHAR(20) DEFAULT 'pending',
    data JSONB DEFAULT '{}',
    files JSONB DEFAULT '[]',
    reviewer_id BIGINT REFERENCES users(id),
    review_comment TEXT,
    reviewed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
                                                           );

CREATE INDEX IF NOT EXISTS idx_admin_submissions_project ON admin_submissions(project_id);
CREATE INDEX IF NOT EXISTS idx_admin_submissions_status ON admin_submissions(status);
CREATE INDEX IF NOT EXISTS idx_admin_submissions_state ON admin_submissions(state_id);
CREATE INDEX IF NOT EXISTS idx_admin_submissions_reviewer ON admin_submissions(reviewer_id);

-- Submission Reviews
CREATE TABLE IF NOT EXISTS admin_submission_reviews (
                                                        id BIGSERIAL PRIMARY KEY,
                                                        submission_id VARCHAR(36) NOT NULL REFERENCES admin_submissions(id) ON DELETE CASCADE,
    reviewer_id BIGINT NOT NULL REFERENCES users(id),
    action VARCHAR(30) NOT NULL,
    comment TEXT,
    grade INT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_submission_reviews_submission ON admin_submission_reviews(submission_id);

-- Supervisor Assignments
CREATE TABLE IF NOT EXISTS admin_supervisor_assignments (
                                                            id BIGSERIAL PRIMARY KEY,
                                                            team_id BIGINT NOT NULL UNIQUE REFERENCES teams(id) ON DELETE CASCADE,
    supervisor_id BIGINT NOT NULL REFERENCES users(id),
    assigned_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_supervisor_assignments_supervisor ON admin_supervisor_assignments(supervisor_id);

-- Activity Log
CREATE TABLE IF NOT EXISTS admin_activities (
                                                id BIGSERIAL PRIMARY KEY,
                                                activity_type VARCHAR(50) NOT NULL,
    description TEXT,
    actor_id BIGINT NOT NULL REFERENCES users(id),
    target_id BIGINT,
    target_type VARCHAR(50),
    metadata JSONB DEFAULT '{}',
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_admin_activities_type ON admin_activities(activity_type);
CREATE INDEX IF NOT EXISTS idx_admin_activities_actor ON admin_activities(actor_id);
CREATE INDEX IF NOT EXISTS idx_admin_activities_target ON admin_activities(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_admin_activities_created ON admin_activities(created_at DESC);
