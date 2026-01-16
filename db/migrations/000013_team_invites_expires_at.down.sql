-- =============================================
-- Migration: 000013_team_invites_expires_at (ROLLBACK)
-- =============================================

DROP INDEX IF EXISTS idx_team_invites_expires_at;

ALTER TABLE team_invites
DROP COLUMN IF EXISTS expires_at;
