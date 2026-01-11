CREATE TABLE IF NOT EXISTS admin_supervisor_requests (
    id VARCHAR(36) PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    project_id BIGINT REFERENCES projects(id),
    supervisor_id BIGINT NOT NULL REFERENCES users(id),
    requested_by BIGINT NOT NULL REFERENCES users(id),
    status VARCHAR(30) DEFAULT 'pending',
    message TEXT,
    proposed_topic VARCHAR(500),
    reject_reason TEXT,
    responded_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS admin_supervisor_request_history (
    id BIGSERIAL PRIMARY KEY,
    request_id VARCHAR(36) NOT NULL REFERENCES admin_supervisor_requests(id) ON DELETE CASCADE,
    action VARCHAR(30) NOT NULL,
    actor_id BIGINT NOT NULL REFERENCES users(id),
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE INDEX idx_supervisor_requests_team ON admin_supervisor_requests(team_id);
CREATE INDEX idx_supervisor_requests_supervisor ON admin_supervisor_requests(supervisor_id);
CREATE INDEX idx_supervisor_requests_status ON admin_supervisor_requests(status);
