CREATE TABLE IF NOT EXISTS admin_pre_defense_submissions (
    id VARCHAR(36) PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    project_id BIGINT NOT NULL REFERENCES projects(id),
    supervisor_id BIGINT REFERENCES users(id),
    submitted_by BIGINT NOT NULL REFERENCES users(id),
    status VARCHAR(30) DEFAULT 'pending',
    scheduled_date TIMESTAMP WITH TIME ZONE,
    scheduled_time VARCHAR(10),
    location VARCHAR(255),
    meeting_link VARCHAR(500),
    duration_minutes INT DEFAULT 30,
    grade INT CHECK (grade >= 0 AND grade <= 100),
    grade_comment TEXT,
    graded_by BIGINT REFERENCES users(id),
    graded_at TIMESTAMP WITH TIME ZONE,
    result VARCHAR(30),
    result_comment TEXT,
    recommendations TEXT[],
    submitted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS admin_pre_defense_commission (
    id BIGSERIAL PRIMARY KEY,
    submission_id VARCHAR(36) NOT NULL REFERENCES admin_pre_defense_submissions(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id),
    role VARCHAR(30) NOT NULL,
    is_present BOOLEAN DEFAULT FALSE,
    individual_grade INT,
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS admin_pre_defense_documents (
    id VARCHAR(36) PRIMARY KEY,
    submission_id VARCHAR(36) NOT NULL REFERENCES admin_pre_defense_submissions(id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL,
    file_type VARCHAR(50),
    display_name VARCHAR(255),
    size BIGINT,
    download_url VARCHAR(500),
    uploaded_by BIGINT REFERENCES users(id),
    uploaded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS admin_pre_defense_history (
    id BIGSERIAL PRIMARY KEY,
    submission_id VARCHAR(36) NOT NULL REFERENCES admin_pre_defense_submissions(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    actor_id BIGINT NOT NULL REFERENCES users(id),
    old_value TEXT,
    new_value TEXT,
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE INDEX idx_pre_defense_team ON admin_pre_defense_submissions(team_id);
CREATE INDEX idx_pre_defense_status ON admin_pre_defense_submissions(status);
CREATE INDEX idx_pre_defense_scheduled ON admin_pre_defense_submissions(scheduled_date);
