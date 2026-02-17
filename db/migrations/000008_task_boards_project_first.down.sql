BEGIN;

-- rollback уникальности по project_id
DROP INDEX IF EXISTS ux_task_boards_project_id;

-- project_id снова nullable
ALTER TABLE task_boards
    ALTER COLUMN project_id DROP NOT NULL;

-- FK project_id обратно в ON DELETE SET NULL (как было в init) [[16]]
ALTER TABLE task_boards
DROP CONSTRAINT IF EXISTS task_boards_project_id_fkey;

ALTER TABLE task_boards
    ADD CONSTRAINT task_boards_project_id_fkey
        FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL;

-- возвращаем уникальность по team_id
ALTER TABLE task_boards
    ADD CONSTRAINT task_boards_team_id_key UNIQUE (team_id);

COMMIT;
