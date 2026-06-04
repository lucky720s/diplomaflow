BEGIN;

-- Hot-path indexes for the supervisor-facing projects module.
--
-- Ownership lookups go through admin_supervisor_assignments by supervisor_id
-- (team_id already has a UNIQUE index, but supervisor_id is unindexed today).
-- The list/detail stats filter admin_submissions by (project_id, status) and
-- read admin_grade_history by project_id.
--
-- NB: idx_admin_submissions_project (project_id) already exists (migration 000025);
-- the composite below additionally serves the status filter.
CREATE INDEX IF NOT EXISTS idx_admin_supervisor_assignments_supervisor ON admin_supervisor_assignments(supervisor_id);
CREATE INDEX IF NOT EXISTS idx_admin_submissions_project_status        ON admin_submissions(project_id, status);
CREATE INDEX IF NOT EXISTS idx_admin_grade_history_project             ON admin_grade_history(project_id);

COMMIT;
