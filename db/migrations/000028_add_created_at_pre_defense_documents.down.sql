BEGIN;

ALTER TABLE admin_pre_defense_documents
    DROP COLUMN IF EXISTS created_at;

COMMIT;
