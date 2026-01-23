-- task_boards: Kanban-доска для команды (одна доска на команду)
CREATE TABLE IF NOT EXISTS task_boards (
                                           id BIGSERIAL PRIMARY KEY,
                                           team_id BIGINT NOT NULL UNIQUE REFERENCES teams(id) ON DELETE CASCADE,
    project_id BIGINT REFERENCES projects(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL DEFAULT 'Доска задач',
    description TEXT,

    -- Настройки доски (JSON)
    settings JSONB DEFAULT '{
        "default_column": "todo",
        "allow_custom_columns": false,
        "show_completed": true,
        "labels": ["backend", "frontend", "docs", "research", "urgent"]
    }',

    created_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
                                                                                                   );

CREATE INDEX idx_task_boards_team ON task_boards(team_id);
CREATE INDEX idx_task_boards_project ON task_boards(project_id);
CREATE INDEX idx_task_boards_deleted_at ON task_boards(deleted_at);

-- task_columns: Колонки на доске (статусы задач)
CREATE TABLE IF NOT EXISTS task_columns (
                                            id BIGSERIAL PRIMARY KEY,
                                            board_id BIGINT NOT NULL REFERENCES task_boards(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) NOT NULL,
    description TEXT,
    color VARCHAR(20) DEFAULT '#6B7280',
    icon VARCHAR(50),
    order_index INT NOT NULL DEFAULT 0,

    -- WIP лимит (Work In Progress limit, 0 = без лимита)
    wip_limit INT DEFAULT 0,

    is_default BOOLEAN DEFAULT FALSE,
    is_done_column BOOLEAN DEFAULT FALSE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(board_id, slug)
    );

CREATE INDEX idx_task_columns_board ON task_columns(board_id);
CREATE INDEX idx_task_columns_order ON task_columns(board_id, order_index);

-- tasks: Задачи
CREATE TABLE IF NOT EXISTS tasks (
    id BIGSERIAL PRIMARY KEY,
    board_id BIGINT NOT NULL REFERENCES task_boards(id) ON DELETE CASCADE,
    column_id BIGINT NOT NULL REFERENCES task_columns(id) ON DELETE RESTRICT,

    -- Основные поля
    title VARCHAR(500) NOT NULL,
    description TEXT,

    -- Статус и приоритет
    status VARCHAR(30) NOT NULL DEFAULT 'todo',
    priority VARCHAR(20) NOT NULL DEFAULT 'medium',

    -- Люди
    assignee_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_by BIGINT NOT NULL REFERENCES users(id),

    -- Сроки
    due_date DATE,
    due_time TIME,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,

    -- Оценка времени (в минутах)
    estimated_minutes INT DEFAULT 0,
    actual_minutes INT DEFAULT 0,

    -- Позиция в колонке (для drag-and-drop сортировки)
    position INT NOT NULL DEFAULT 0,

    workflow_step_id BIGINT REFERENCES states(id) ON DELETE SET NULL,

    -- Метаданные
    labels JSONB DEFAULT '[]',
    custom_fields JSONB DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_tasks_board ON tasks(board_id);
CREATE INDEX idx_tasks_column ON tasks(column_id);
CREATE INDEX idx_tasks_assignee ON tasks(assignee_id);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_priority ON tasks(priority);
CREATE INDEX idx_tasks_due_date ON tasks(due_date) WHERE due_date IS NOT NULL;
CREATE INDEX idx_tasks_deleted_at ON tasks(deleted_at);
CREATE INDEX idx_tasks_position ON tasks(column_id, position);
CREATE INDEX idx_tasks_created_by ON tasks(created_by);

-- task_comments: Комментарии к задачам
CREATE TABLE IF NOT EXISTS task_comments (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    author_id BIGINT NOT NULL REFERENCES users(id),

    content TEXT NOT NULL,

    -- Упоминания пользователей (@user)
    mentions JSONB DEFAULT '[]',

    -- Редактирование
    edited_at TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
                                                                                              );

CREATE INDEX idx_task_comments_task ON task_comments(task_id);
CREATE INDEX idx_task_comments_author ON task_comments(author_id);
CREATE INDEX idx_task_comments_created ON task_comments(task_id, created_at DESC);

-- task_attachments: Вложения к задачам
CREATE TABLE IF NOT EXISTS task_attachments (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,

    file_id VARCHAR(36) REFERENCES file_metadata(id) ON DELETE SET NULL,
    file_name VARCHAR(255) NOT NULL,
    file_type VARCHAR(100),
    file_size BIGINT DEFAULT 0,

    uploaded_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE INDEX idx_task_attachments_task ON task_attachments(task_id);

-- task_activity_log: История изменений задач (audit trail)
CREATE TABLE IF NOT EXISTS task_activity_log (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    actor_id BIGINT NOT NULL REFERENCES users(id),

    action VARCHAR(50) NOT NULL,
    field_name VARCHAR(100),
    old_value TEXT,
    new_value TEXT,

    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE INDEX idx_task_activity_task ON task_activity_log(task_id);
CREATE INDEX idx_task_activity_created ON task_activity_log(task_id, created_at DESC);
CREATE INDEX idx_task_activity_actor ON task_activity_log(actor_id);

-- task_watchers: Наблюдатели задачи (получают уведомления)
CREATE TABLE IF NOT EXISTS task_watchers (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(task_id, user_id)
    );

CREATE INDEX idx_task_watchers_task ON task_watchers(task_id);
CREATE INDEX idx_task_watchers_user ON task_watchers(user_id);


CREATE OR REPLACE FUNCTION create_task_board_for_team()
RETURNS TRIGGER AS $$
DECLARE
v_board_id BIGINT;
    v_leader_id BIGINT;
BEGIN
    -- Находим лидера команды
SELECT user_id INTO v_leader_id
FROM team_members
WHERE team_id = NEW.id AND role = 'leader'
    LIMIT 1;

-- Если лидер не найден, используем первого участника
IF v_leader_id IS NULL THEN
SELECT user_id INTO v_leader_id
FROM team_members
WHERE team_id = NEW.id
    LIMIT 1;
END IF;

    -- Если всё ещё нет - пропускаем (команда без участников)
    IF v_leader_id IS NULL THEN
        RETURN NEW;
END IF;

    -- Создаём доску
INSERT INTO task_boards (team_id, project_id, name, created_by)
VALUES (NEW.id, NEW.project_id, 'Задачи команды: ' || NEW.name, v_leader_id)
    RETURNING id INTO v_board_id;

-- Создаём стандартные колонки
INSERT INTO task_columns (board_id, name, slug, color, order_index, is_default, is_done_column) VALUES
    (v_board_id, 'К выполнению', 'todo', '#6B7280', 0, TRUE, FALSE),
    (v_board_id, 'В работе', 'in_progress', '#3B82F6', 1, FALSE, FALSE),
    (v_board_id, 'На проверке', 'review', '#F59E0B', 2, FALSE, FALSE),
    (v_board_id, 'Готово', 'done', '#10B981', 3, FALSE, TRUE);

RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Триггер (опционально - можно создавать доску вручную)
-- CREATE TRIGGER trg_create_task_board_after_team
--     AFTER INSERT ON teams
--     FOR EACH ROW
--     EXECUTE FUNCTION create_task_board_for_team();
