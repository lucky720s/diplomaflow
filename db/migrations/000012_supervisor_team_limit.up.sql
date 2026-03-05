BEGIN;

-- Индивидуальный лимит команд для каждого преподавателя (supervisor)
-- Если max_teams = 0 — используется дефолт из workflow settings
CREATE TABLE IF NOT EXISTS supervisor_settings (
                                                   id          BIGSERIAL PRIMARY KEY,
                                                   user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    department_id BIGINT NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    max_teams   INT NOT NULL DEFAULT 0,  -- 0 = использовать дефолт из workflow/department config
    updated_by  BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, department_id)
    );

CREATE INDEX IF NOT EXISTS idx_supervisor_settings_user
    ON supervisor_settings(user_id);
CREATE INDEX IF NOT EXISTS idx_supervisor_settings_dept
    ON supervisor_settings(department_id);

COMMIT;
