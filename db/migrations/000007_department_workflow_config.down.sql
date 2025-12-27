-- db/migrations/000007_department_workflow_config.down.sql
DROP TABLE IF EXISTS department_custom_steps CASCADE;
DROP TABLE IF EXISTS department_workflow_configs CASCADE;
DROP TABLE IF EXISTS action_registry CASCADE;
