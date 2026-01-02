-- =============================================
-- Migration: 000008_topic_registrations
-- Description: Таблица заявлений на регистрацию дипломной темы
-- =============================================

-- Topic Registrations (заявления на регистрацию темы)
CREATE TABLE IF NOT EXISTS admin_topic_registrations (
                                                         id VARCHAR(36) PRIMARY KEY,
    team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    project_id BIGINT REFERENCES projects(id) ON DELETE SET NULL,
    proposed_topic VARCHAR(500) NOT NULL,
    topic_description TEXT,
    supervisor_id BIGINT NOT NULL REFERENCES users(id),
    submitted_by BIGINT NOT NULL REFERENCES users(id),
    status VARCHAR(30) DEFAULT 'pending',
    rejection_reason TEXT,
    comment TEXT,
    reviewer_id BIGINT REFERENCES users(id),
    reviewed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
                                                     );

CREATE INDEX IF NOT EXISTS idx_topic_registrations_team ON admin_topic_registrations(team_id);
CREATE INDEX IF NOT EXISTS idx_topic_registrations_status ON admin_topic_registrations(status);
CREATE INDEX IF NOT EXISTS idx_topic_registrations_supervisor ON admin_topic_registrations(supervisor_id);

-- Topic Registration Reviews (история проверок заявления на тему)
CREATE TABLE IF NOT EXISTS admin_topic_registration_reviews (
                                                                id BIGSERIAL PRIMARY KEY,
                                                                registration_id VARCHAR(36) NOT NULL REFERENCES admin_topic_registrations(id) ON DELETE CASCADE,
    reviewer_id BIGINT NOT NULL REFERENCES users(id),
    action VARCHAR(30) NOT NULL,
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_topic_reg_reviews_registration ON admin_topic_registration_reviews(registration_id);

-- Убираем letter_grade из admin_grades (переходим на чисто балльную систему)
ALTER TABLE admin_grades DROP COLUMN IF EXISTS letter_grade;
