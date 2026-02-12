BEGIN;

-- Откат: возвращаем nullable project_id и NOT NULL team_id.
-- Перед SET NOT NULL — backfill team_id из projects.team_id на случай, если появились NULL.

-- admin_topic_registrations
UPDATE admin_topic_registrations tr
SET team_id = p.team_id
    FROM projects p
WHERE tr.team_id IS NULL
  AND tr.project_id = p.id;

ALTER TABLE admin_topic_registrations
    ALTER COLUMN team_id SET NOT NULL;

DROP INDEX IF EXISTS idx_topic_registrations_project;

ALTER TABLE admin_topic_registrations
    ALTER COLUMN project_id DROP NOT NULL;


-- admin_supervisor_requests
UPDATE admin_supervisor_requests sr
SET team_id = p.team_id
    FROM projects p
WHERE sr.team_id IS NULL
  AND sr.project_id = p.id;

ALTER TABLE admin_supervisor_requests
    ALTER COLUMN team_id SET NOT NULL;

DROP INDEX IF EXISTS idx_supervisor_requests_project;

ALTER TABLE admin_supervisor_requests
    ALTER COLUMN project_id DROP NOT NULL;

COMMIT;
