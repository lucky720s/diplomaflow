BEGIN;

-- Синхронизируем существующие проекты
UPDATE projects
SET status = current_state_name,
    updated_at = NOW()
WHERE status = 'active'
  AND current_state_name IS NOT NULL
  AND current_state_name != '';

-- Добавляем поле topic_registered_at (для задачи 2)
ALTER TABLE projects ADD COLUMN IF NOT EXISTS topic_registered_at TIMESTAMPTZ;

COMMIT;
