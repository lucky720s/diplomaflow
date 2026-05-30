BEGIN;

ALTER TABLE file_metadata
    DROP COLUMN IF EXISTS mime_type;

COMMIT;
