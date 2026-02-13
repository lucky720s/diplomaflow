BEGIN;

DO $$
DECLARE
v_workflow_id BIGINT;
BEGIN
SELECT id INTO v_workflow_id
FROM workflows
WHERE name = 'Дипломный проект (ПИ)'
    LIMIT 1;

IF v_workflow_id IS NULL THEN
    RETURN;
END IF;

DELETE FROM department_custom_steps
WHERE department_config_id IN (SELECT id FROM department_workflow_configs WHERE workflow_id = v_workflow_id);

DELETE FROM department_workflow_configs WHERE workflow_id = v_workflow_id;
DELETE FROM state_actions WHERE state_id IN (SELECT id FROM states WHERE workflow_id = v_workflow_id);
DELETE FROM transitions WHERE workflow_id = v_workflow_id;
DELETE FROM states WHERE workflow_id = v_workflow_id;
DELETE FROM workflows WHERE id = v_workflow_id;
END $$;

COMMIT;
