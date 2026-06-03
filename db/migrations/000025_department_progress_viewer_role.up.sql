BEGIN;

-- Department-scoped role that lets a teacher view department-wide team progress
-- (read-only). Uses the existing dynamic-role mechanism (roles + user_role_assignments,
-- slug stored in roles.name) exactly like 'norm_control' / 'antiplagiat' / 'commission'.
-- Idempotent: re-running is a no-op.
INSERT INTO roles (name, display_name, department_id)
SELECT 'department_progress_viewer', 'Department Progress Viewer', d.id
FROM departments d
WHERE d.deleted_at IS NULL
ON CONFLICT (name, department_id) DO NOTHING;

-- Hot-path indexes for the batch progress aggregation (project_id is FK-only today).
CREATE INDEX IF NOT EXISTS idx_admin_pre_defense_submissions_project ON admin_pre_defense_submissions(project_id);
CREATE INDEX IF NOT EXISTS idx_admin_topic_registrations_project     ON admin_topic_registrations(project_id);
CREATE INDEX IF NOT EXISTS idx_admin_submissions_project             ON admin_submissions(project_id);

COMMIT;
