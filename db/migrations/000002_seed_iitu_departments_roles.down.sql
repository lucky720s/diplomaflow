BEGIN;

DELETE FROM action_registry
WHERE id IN ('SEND_NOTIFICATION', 'LOCK_TEAM_COMPOSITION', 'CHECK_ANTIPLAGIAT');

DO $$
DECLARE
v_iitu_id BIGINT;
BEGIN
SELECT id INTO v_iitu_id FROM universities WHERE short_name = 'IITU' LIMIT 1;
IF v_iitu_id IS NULL THEN
    RETURN;
END IF;

DELETE FROM roles WHERE department_id IN (SELECT id FROM departments WHERE university_id = v_iitu_id);
DELETE FROM departments WHERE university_id = v_iitu_id;
DELETE FROM universities WHERE id = v_iitu_id;
END $$;

COMMIT;
