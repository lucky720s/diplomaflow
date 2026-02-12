BEGIN;

-- =========================
-- admin_topic_registrations: project-first
-- В 000008: team_id NOT NULL, project_id nullable [[11]]
-- =========================

-- 1) backfill project_id через новую связь projects.team_id = admin_topic_registrations.team_id [[11]]
UPDATE admin_topic_registrations tr
SET project_id = p.id
FROM projects p
WHERE tr.project_id IS NULL
AND p.team_id = tr.team_id;

-- 2) теперь project_id обязателен
ALTER TABLE admin_topic_registrations
ALTER COLUMN project_id SET NOT NULL;

-- 3) team_id делаем optional (для гибкости Variant B)
ALTER TABLE admin_topic_registrations
ALTER COLUMN team_id DROP NOT NULL;

-- 4) индекс по project_id (часто будем фильтровать по проекту)
CREATE INDEX IF NOT EXISTS idx_topic_registrations_project
ON admin_topic_registrations(project_id);


-- =========================
-- admin_supervisor_requests: project-first
-- В 000009: team_id NOT NULL, project_id nullable [[11]]
-- =========================

-- 1) backfill project_id
UPDATE admin_supervisor_requests sr
SET project_id = p.id
FROM projects p
WHERE sr.project_id IS NULL
AND p.team_id = sr.team_id;

-- 2) project_id обязателен
ALTER TABLE admin_supervisor_requests
ALTER COLUMN project_id SET NOT NULL;

-- 3) team_id optional
ALTER TABLE admin_supervisor_requests
ALTER COLUMN team_id DROP NOT NULL;

-- 4) индекс по project_id
CREATE INDEX IF NOT EXISTS idx_supervisor_requests_project
ON admin_supervisor_requests(project_id);

COMMIT;
