BEGIN;

DROP INDEX IF EXISTS ux_workflows_active_dept;
DROP INDEX IF EXISTS idx_projects_deadline;
DROP INDEX IF EXISTS idx_projects_status;
DROP INDEX IF EXISTS idx_projects_team_id;
DROP INDEX IF EXISTS idx_projects_dept_wf_state;
DROP INDEX IF EXISTS idx_state_histories_direction;
DROP INDEX IF EXISTS idx_state_histories_project_created;

ALTER TABLE state_histories
    DROP COLUMN IF EXISTS actor_role,
    DROP COLUMN IF EXISTS direction,
    DROP COLUMN IF EXISTS transition_id;

COMMIT;
