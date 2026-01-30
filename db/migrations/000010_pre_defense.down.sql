-- Drop indexes first
DROP INDEX IF EXISTS idx_pre_defense_scheduled;
DROP INDEX IF EXISTS idx_pre_defense_status;
DROP INDEX IF EXISTS idx_pre_defense_team;

-- Drop tables in reverse order (dependencies first)
DROP TABLE IF EXISTS admin_pre_defense_history;
DROP TABLE IF EXISTS admin_pre_defense_documents;
DROP TABLE IF EXISTS admin_pre_defense_commission;
DROP TABLE IF EXISTS admin_pre_defense_submissions;
