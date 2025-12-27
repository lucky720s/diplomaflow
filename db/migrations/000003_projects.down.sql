-- =============================================
-- Migration: 000003_projects (ROLLBACK)
-- =============================================

DROP TABLE IF EXISTS form_submissions CASCADE;
DROP TABLE IF EXISTS outbox_events CASCADE;
DROP TABLE IF EXISTS state_histories CASCADE;

ALTER TABLE file_metadata DROP CONSTRAINT IF EXISTS fk_file_metadata_project;
ALTER TABLE teams DROP CONSTRAINT IF EXISTS fk_teams_project;

DROP TABLE IF EXISTS projects CASCADE;
