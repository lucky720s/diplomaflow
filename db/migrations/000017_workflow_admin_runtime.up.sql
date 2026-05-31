BEGIN;

-- ================================================================
-- 017: Workflow admin runtime support
-- ================================================================
-- 1. Extend state_histories with transition tracking columns
-- 2. Add useful indices on projects and state_histories
-- ================================================================

-- Добавляем колонки в state_histories для аудита admin-действий
ALTER TABLE state_histories
    ADD COLUMN IF NOT EXISTS transition_id BIGINT NULL,
    ADD COLUMN IF NOT EXISTS direction     VARCHAR(20) NULL,
    ADD COLUMN IF NOT EXISTS actor_role    VARCHAR(100) NULL;

-- Индексы для state_histories (быстрые выборки истории проекта)
CREATE INDEX IF NOT EXISTS idx_state_histories_project_created
    ON state_histories(project_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_state_histories_direction
    ON state_histories(direction)
    WHERE direction IS NOT NULL;

-- Индексы для projects (мониторинг по workflow/кафедре/стэйту)
CREATE INDEX IF NOT EXISTS idx_projects_dept_wf_state
    ON projects(department_id, workflow_id, current_state_id);

CREATE INDEX IF NOT EXISTS idx_projects_team_id
    ON projects(team_id)
    WHERE team_id > 0;

CREATE INDEX IF NOT EXISTS idx_projects_status
    ON projects(status);

CREATE INDEX IF NOT EXISTS idx_projects_deadline
    ON projects(deadline_at)
    WHERE deadline_at IS NOT NULL;

-- Unique partial index: один активный workflow на кафедру
-- (у workflow уже есть is_active, но не было уникального constraint)
CREATE UNIQUE INDEX IF NOT EXISTS ux_workflows_active_dept
    ON workflows(department_id)
    WHERE is_active = TRUE AND deleted_at IS NULL;

COMMIT;
