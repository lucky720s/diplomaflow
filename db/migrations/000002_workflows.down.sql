-- =============================================
-- Migration: 000002_workflows (ROLLBACK)
-- =============================================

DROP TABLE IF EXISTS workflow_templates CASCADE;
DROP TABLE IF EXISTS state_conditions CASCADE;
DROP TABLE IF EXISTS state_actions CASCADE;
DROP TABLE IF EXISTS transitions CASCADE;
DROP TABLE IF EXISTS states CASCADE;
DROP TABLE IF EXISTS workflows CASCADE;
