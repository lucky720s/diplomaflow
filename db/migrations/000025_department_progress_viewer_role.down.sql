BEGIN;

DROP INDEX IF EXISTS idx_admin_submissions_project;
DROP INDEX IF EXISTS idx_admin_topic_registrations_project;
DROP INDEX IF EXISTS idx_admin_pre_defense_submissions_project;

-- Removing the role rows cascades to user_role_assignments (role_id ON DELETE CASCADE).
DELETE FROM roles WHERE name = 'department_progress_viewer';

COMMIT;
