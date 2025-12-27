-- =============================================
-- Migration: 000001_initial_schema (ROLLBACK)
-- =============================================

DROP TABLE IF EXISTS file_metadata CASCADE;
DROP TABLE IF EXISTS notifications CASCADE;
DROP TABLE IF EXISTS team_invites CASCADE;
DROP TABLE IF EXISTS team_members CASCADE;
DROP TABLE IF EXISTS teams CASCADE;
DROP TABLE IF EXISTS refresh_tokens CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS roles CASCADE;
DROP TABLE IF EXISTS departments CASCADE;
DROP TABLE IF EXISTS universities CASCADE;
