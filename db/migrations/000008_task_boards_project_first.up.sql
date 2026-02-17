BEGIN;

-- 1) Попытаться проставить project_id для существующих досок по team_id
-- (в вашей схеме projects.team_id уникален) [[16]]
UPDATE task_boards b
SET project_id = p.id
    FROM projects p
WHERE p.team_id = b.team_id
  AND b.project_id IS NULL;

-- 2) Удаляем уникальность по team_id (раньше было: team_id BIGINT NOT NULL UNIQUE) [[16]]
ALTER TABLE task_boards
DROP CONSTRAINT IF EXISTS task_boards_team_id_key;

-- 3) Меняем FK project_id: раньше ON DELETE SET NULL, но теперь project_id будет NOT NULL
-- => делаем ON DELETE CASCADE (если проект удалили, доску тоже) [[16]]
ALTER TABLE task_boards
DROP CONSTRAINT IF EXISTS task_boards_project_id_fkey;

ALTER TABLE task_boards
    ADD CONSTRAINT task_boards_project_id_fkey
        FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE;

-- 4) project_id теперь обязателен
ALTER TABLE task_boards
    ALTER COLUMN project_id SET NOT NULL;

-- 5) 1 board на 1 project (учитываем soft-delete)
CREATE UNIQUE INDEX IF NOT EXISTS ux_task_boards_project_id
    ON task_boards(project_id)
    WHERE deleted_at IS NULL;

-- 6) team_id оставляем как обычный индекс (для joins/фильтраций)
CREATE INDEX IF NOT EXISTS idx_task_boards_team_id
    ON task_boards(team_id);

COMMIT;
