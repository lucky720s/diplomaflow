-- Удаляем demo workflow
DO $$
DECLARE
v_workflow_id BIGINT;
BEGIN
SELECT id INTO v_workflow_id
FROM workflows
WHERE name = 'Дипломный проект 2025'
    LIMIT 1;

IF v_workflow_id IS NOT NULL THEN
DELETE FROM state_actions WHERE state_id IN (SELECT id FROM states WHERE workflow_id = v_workflow_id);
DELETE FROM transitions WHERE workflow_id = v_workflow_id;
DELETE FROM states WHERE workflow_id = v_workflow_id;
DELETE FROM workflows WHERE id = v_workflow_id;
END IF;
END $$;
