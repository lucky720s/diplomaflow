DROP INDEX IF EXISTS idx_teams_department_id;
DROP INDEX IF EXISTS ux_teams_university_invite_code;

ALTER TABLE teams
DROP COLUMN IF EXISTS composition_locked_at,
  DROP COLUMN IF EXISTS composition_locked,
  DROP COLUMN IF EXISTS invite_code,
  DROP COLUMN IF EXISTS department_id,
  DROP COLUMN IF EXISTS university_id;
