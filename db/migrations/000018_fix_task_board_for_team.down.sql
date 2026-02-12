CREATE OR REPLACE FUNCTION create_task_board_for_team()
RETURNS TRIGGER AS $$
DECLARE
v_board_id  BIGINT;
    v_leader_id BIGINT;
BEGIN
SELECT user_id INTO v_leader_id
FROM team_members
WHERE team_id = NEW.id AND role = 'leader'
    LIMIT 1;

IF v_leader_id IS NULL THEN
SELECT user_id INTO v_leader_id
FROM team_members
WHERE team_id = NEW.id
    LIMIT 1;
END IF;

    IF v_leader_id IS NULL THEN
        RETURN NEW;
END IF;

INSERT INTO task_boards (team_id, project_id, name, created_by)
VALUES (NEW.id, NULL, 'Задачи команды: ' || NEW.name, v_leader_id)
    RETURNING id INTO v_board_id;

INSERT INTO task_columns (board_id, name, slug, color, order_index, is_default, is_done_column) VALUES
                                                                                                    (v_board_id, 'К выполнению', 'todo', '#6B7280', 0, TRUE,  FALSE),
                                                                                                    (v_board_id, 'В работе',     'in_progress', '#3B82F6', 1, FALSE, FALSE),
                                                                                                    (v_board_id, 'На проверке',  'review', '#F59E0B', 2, FALSE, FALSE),
                                                                                                    (v_board_id, 'Готово',       'done', '#10B981', 3, FALSE, TRUE);

RETURN NEW;
END;
$$ LANGUAGE plpgsql;
