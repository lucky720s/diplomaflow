BEGIN;
UPDATE roles
SET name = display_name
WHERE display_name IS NOT NULL;

ALTER TABLE roles
DROP COLUMN IF EXISTS display_name;

COMMIT;
