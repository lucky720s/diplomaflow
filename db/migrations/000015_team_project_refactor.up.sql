-- 1) backfill: перенести связь из teams.project_id в projects.team_id
UPDATE projects p
SET team_id = t.id
    FROM teams t
WHERE t.project_id = p.id
  AND (p.team_id IS NULL OR p.team_id = 0);
-- 2) drop fk (имя в 000003: fk_teams_project) [[1]]
ALTER TABLE teams DROP CONSTRAINT IF EXISTS fk_teams_project;

-- 3) drop column
ALTER TABLE teams DROP COLUMN IF EXISTS project_id;

CREATE UNIQUE INDEX IF NOT EXISTS ux_projects_team_id
    ON projects(team_id)
    WHERE team_id IS NOT NULL;
