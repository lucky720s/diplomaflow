-- Удаляем триггер (если был создан)
DROP TRIGGER IF EXISTS trg_create_task_board_after_team ON teams;
DROP FUNCTION IF EXISTS create_task_board_for_team();

-- Удаляем таблицы в правильном порядке (из-за FK)
DROP TABLE IF EXISTS task_watchers CASCADE;
DROP TABLE IF EXISTS task_activity_log CASCADE;
DROP TABLE IF EXISTS task_attachments CASCADE;
DROP TABLE IF EXISTS task_comments CASCADE;
DROP TABLE IF EXISTS tasks CASCADE;
DROP TABLE IF EXISTS task_columns CASCADE;
DROP TABLE IF EXISTS task_boards CASCADE;
