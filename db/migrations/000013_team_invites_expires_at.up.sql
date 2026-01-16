-- =============================================
-- Migration: 000013_team_invites_expires_at
-- Description: Add expires_at column to team_invites (required by service model)
-- =============================================

ALTER TABLE team_invites
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP WITH TIME ZONE;

-- Backfill for existing rows (default 3 days from created_at)
UPDATE team_invites
SET expires_at = COALESCE(expires_at, created_at + INTERVAL '3 days')
WHERE expires_at IS NULL;

ALTER TABLE team_invites
    ALTER COLUMN expires_at SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_team_invites_expires_at ON team_invites(expires_at);
