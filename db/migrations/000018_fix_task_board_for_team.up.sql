-- Fix create_task_board_for_team(): teams.project_id was dropped in 000015, use projects.team_id instead [[11]].

CREATE OR REPLACE FUNCTION create_task_board_for_team()
RETURNS TRIGGER AS $$
DECLARE
v_board_id   BIGINT;
    v_leader_id  BIGINT;
    v_project_id BIGINT;
BEGIN
    -- Find leader
SELECT user_id INTO v_leader_id
FROM team_members
WHERE team_id = NEW.id AND role = 'leader'
    LIMIT 1;

-- Fallback: any member
IF v_leader_id IS NULL THEN
SELECT user_id INTO v_leader_id
FROM team_members
WHERE team_id = NEW.id
    LIMIT 1;
END IF;

    -- If still no members - do nothing
    IF v_leader_id IS NULL THEN
        RETURN NEW;
END IF;

    -- Resolve project_id via new relation: projects.team_id = teams.id [[11]]
SELECT id INTO v_project_id
FROM projects
WHERE team_id = NEW.id
ORDER BY id DESC
    LIMIT 1;

-- Create board (project_id может быть NULL, это ок)
INSERT INTO task_boards (team_id, project_id, name, created_by)
VALUES (NEW.id, v_project_id, 'Задачи команды: ' || NEW.name, v_leader_id)
    RETURNING id INTO v_board_id;

-- Default columns
INSERT INTO task_columns (board_id, name, slug, color, order_index, is_default, is_done_column) VALUES
                                                                                                    (v_board_id, 'К выполнению', 'todo', '#6B7280', 0, TRUE,  FALSE),
                                                                                                    (v_board_id, 'В работе',     'in_progress', '#3B82F6', 1, FALSE, FALSE),
                                                                                                    (v_board_id, 'На проверке',  'review', '#F59E0B', 2, FALSE, FALSE),
                                                                                                    (v_board_id, 'Готово',       'done', '#10B981', 3, FALSE, TRUE);

RETURN NEW;
END;
$$ LANGUAGE plpgsql;
