CREATE UNIQUE INDEX IF NOT EXISTS idx_team_invites_unique_pending
    ON team_invites(team_id, user_id)
    WHERE status = 'PENDING';

CREATE INDEX IF NOT EXISTS idx_team_invites_user_status
    ON team_invites(user_id, status);

CREATE INDEX IF NOT EXISTS idx_team_invites_team_status
    ON team_invites(team_id, status);

CREATE UNIQUE INDEX IF NOT EXISTS idx_team_members_unique_user
    ON team_members(user_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_team_members_unique_team_user
    ON team_members(team_id, user_id);