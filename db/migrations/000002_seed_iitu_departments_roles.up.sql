BEGIN;

-- University: only IITU
INSERT INTO universities (name, short_name)
VALUES ('International Information Technology University', 'IITU')
    ON CONFLICT (short_name) DO NOTHING;

-- Departments: PI + SIB
DO $$
DECLARE
v_iitu_id BIGINT;
BEGIN
SELECT id INTO v_iitu_id FROM universities WHERE short_name = 'IITU' LIMIT 1;
IF v_iitu_id IS NULL THEN
    RAISE EXCEPTION 'IITU not found';
END IF;

INSERT INTO departments (name, university_id)
VALUES
    ('Программная Инженерия', v_iitu_id),
    ('СИБ', v_iitu_id)
    ON CONFLICT (name, university_id) DO NOTHING;
END $$;

-- Role directory for PI (dynamic roles/capabilities for workflow)
DO $$
DECLARE
v_iitu_id BIGINT;
  v_pi_id BIGINT;
BEGIN
SELECT id INTO v_iitu_id FROM universities WHERE short_name = 'IITU' LIMIT 1;
SELECT id INTO v_pi_id FROM departments WHERE university_id = v_iitu_id AND name = 'Программная Инженерия' LIMIT 1;

IF v_pi_id IS NULL THEN
    RAISE EXCEPTION 'PI department not found';
END IF;

INSERT INTO roles (name, department_id) VALUES
                                            ('Professor', v_pi_id),
                                            ('Senior Lecturer', v_pi_id),
                                            ('Associate Professor', v_pi_id),
                                            ('Supervisor', v_pi_id),
                                            ('Norm Control Reviewer', v_pi_id),
                                            ('Economics Reviewer', v_pi_id),
                                            ('Commission Member', v_pi_id),
                                            ('Reviewer of topic', v_pi_id)
    ON CONFLICT (name, department_id) DO NOTHING;
END $$;

-- Workflow action registry (system actions used by workflow engine/plugins)
INSERT INTO action_registry (id, name, description, category, config_schema, is_system, is_enabled)
VALUES
    ('SEND_NOTIFICATION', 'Send notification', 'Creates a notification for a user', 'notification',
     '{"type":"object","properties":{"title":{"type":"string"},"message":{"type":"string"},"recipients":{"type":"array"}}}', TRUE, TRUE),
    ('LOCK_TEAM_COMPOSITION', 'Lock team composition', 'Locks team membership', 'workflow',
     '{"type":"object","properties":{"reason":{"type":"string"}}}', TRUE, TRUE),
    ('CHECK_ANTIPLAGIAT', 'Anti-plagiarism check', 'External plagiarism check', 'external',
     '{"type":"object","properties":{"min_score":{"type":"number"},"service_url":{"type":"string"}}}', TRUE, TRUE)
    ON CONFLICT (id) DO NOTHING;

COMMIT;
