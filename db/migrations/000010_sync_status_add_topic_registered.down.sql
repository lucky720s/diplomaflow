BEGIN;

UPDATE projects
SET status = 'active',
    updated_at = NOW()
WHERE status NOT IN ('active', 'completed', 'cancelled', 'archived');

ALTER TABLE projects DROP COLUMN IF EXISTS topic_registered_at;

COMMIT;
