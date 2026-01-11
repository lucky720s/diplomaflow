-- =============================================
-- Migration: 000009_supervisor_requests (ROLLBACK)
-- =============================================

DROP INDEX IF EXISTS idx_supervisor_requests_status;
DROP INDEX IF EXISTS idx_supervisor_requests_supervisor;
DROP INDEX IF EXISTS idx_supervisor_requests_team;

DROP TABLE IF EXISTS admin_supervisor_request_history CASCADE;
DROP TABLE IF EXISTS admin_supervisor_requests CASCADE;
