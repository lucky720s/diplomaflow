BEGIN;

UPDATE admin_activities
SET metadata = '{}'::jsonb
WHERE metadata IS NULL;

ALTER TABLE admin_activities
    ALTER COLUMN metadata SET DEFAULT '{}'::jsonb;

ALTER TABLE admin_activities
    ALTER COLUMN metadata SET NOT NULL;

COMMIT;