BEGIN;

DROP INDEX IF EXISTS idx_admin_grade_history_project;
DROP INDEX IF EXISTS idx_admin_submissions_project_status;
DROP INDEX IF EXISTS idx_admin_supervisor_assignments_supervisor;

COMMIT;
