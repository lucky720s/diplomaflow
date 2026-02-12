DO $$
DECLARE
v_dept_id BIGINT;
    v_workflow_id BIGINT;

    v_team BIGINT;
    v_sup BIGINT;
    v_topic BIGINT;

BEGIN
    -- Prefer CS/IITU if exists, otherwise use any department
SELECT d.id INTO v_dept_id
FROM departments d
         JOIN universities u ON u.id = d.university_id
WHERE d.name = 'Computer Science' AND u.short_name = 'IITU'
    LIMIT 1;

IF v_dept_id IS NULL THEN
SELECT id INTO v_dept_id FROM departments ORDER BY id LIMIT 1;
END IF;

    IF v_dept_id IS NULL THEN
        RAISE NOTICE 'No departments found, skipping demo workflow seed';
        RETURN;
END IF;

    -- Create workflow if missing
INSERT INTO workflows (name, description, department_id, version, is_active, settings)
VALUES (
           'Diploma Project Demo',
           'Demo workflow for diploma project',
           v_dept_id,
           1,
           TRUE,
           '{"team_required": true, "min_team_size": 1, "max_team_size": 4, "allow_solo_project": true}'::jsonb
       )
    ON CONFLICT (name, department_id, version) DO NOTHING;

SELECT id INTO v_workflow_id
FROM workflows
WHERE name = 'Diploma Project Demo' AND department_id = v_dept_id AND version = 1
    LIMIT 1;

IF v_workflow_id IS NULL THEN
        RAISE EXCEPTION 'Workflow not found/created';
END IF;

    -- ===== STATES (minimal set for your flow) =====

    -- TEAM_FORMATION
INSERT INTO states (workflow_id, name, display_name, description, order_index, type, is_initial, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'TEAM_FORMATION',
           'Team formation',
           'Create or join a team',
           1,
           'TEAM_FORMATION',
           TRUE,
           14,
           '{"team_config":{"min_size":1,"max_size":4,"allow_solo":true,"require_leader":true}}'::jsonb,
           '#3B82F6',
           'users'
       )
    ON CONFLICT DO NOTHING;

SELECT id INTO v_team FROM states
WHERE workflow_id = v_workflow_id AND name = 'TEAM_FORMATION'
    LIMIT 1;

-- SUPERVISOR_SELECTION
INSERT INTO states (workflow_id, name, display_name, description, order_index, type, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'SUPERVISOR_SELECTION',
           'Supervisor selection',
           'Choose a supervisor',
           2,
           'SUPERVISOR_SELECTION',
           7,
           '{"supervisor_config":{"allowed_roles":["teacher","professor"],"max_students_per_supervisor":5}}'::jsonb,
           '#8B5CF6',
           'user-check'
       )
    ON CONFLICT DO NOTHING;

SELECT id INTO v_sup FROM states
WHERE workflow_id = v_workflow_id AND name = 'SUPERVISOR_SELECTION'
    LIMIT 1;

-- TOPIC_APPROVAL
INSERT INTO states (workflow_id, name, display_name, description, order_index, type, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'TOPIC_APPROVAL',
           'Topic approval',
           'Topic must be approved by department',
           3,
           'APPROVAL',
           5,
           '{"review_config":{"reviewer_roles":["teacher","head_of_department"],"min_reviewers":1,"require_comment":false}}'::jsonb,
           '#F59E0B',
           'check-circle'
       )
    ON CONFLICT DO NOTHING;

SELECT id INTO v_topic FROM states
WHERE workflow_id = v_workflow_id AND name = 'TOPIC_APPROVAL'
    LIMIT 1;

IF v_team IS NULL OR v_sup IS NULL OR v_topic IS NULL THEN
        RAISE EXCEPTION 'State IDs missing (team=% sup=% topic=%)', v_team, v_sup, v_topic;
END IF;

    -- ===== TRANSITIONS =====
    -- TEAM_FORMED
INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color)
SELECT v_workflow_id, 'TEAM_FORMED', 'Team formed', v_team, v_sup, 'Team is ready', 'success'
    WHERE NOT EXISTS (
        SELECT 1 FROM transitions
        WHERE workflow_id = v_workflow_id AND event_name = 'TEAM_FORMED'
          AND from_state_id = v_team AND to_state_id = v_sup
    );

-- SUPERVISOR_SELECTED
INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color)
SELECT v_workflow_id, 'SUPERVISOR_SELECTED', 'Supervisor selected', v_sup, v_topic, 'Send for approval', 'primary'
    WHERE NOT EXISTS (
        SELECT 1 FROM transitions
        WHERE workflow_id = v_workflow_id AND event_name = 'SUPERVISOR_SELECTED'
          AND from_state_id = v_sup AND to_state_id = v_topic
    );

-- TOPIC_APPROVED (dummy next step: stay in same state or create next state later)
INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color)
SELECT v_workflow_id, 'TOPIC_APPROVED', 'Topic approved', v_topic, v_topic, 'Approve', 'success'
    WHERE NOT EXISTS (
        SELECT 1 FROM transitions
        WHERE workflow_id = v_workflow_id AND event_name = 'TOPIC_APPROVED'
          AND from_state_id = v_topic AND to_state_id = v_topic
    );

-- TOPIC_REJECTED -> back to SUPERVISOR_SELECTION
INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color, confirm_text)
SELECT v_workflow_id, 'TOPIC_REJECTED', 'Topic rejected', v_topic, v_sup, 'Reject', 'danger', 'Are you sure?'
    WHERE NOT EXISTS (
        SELECT 1 FROM transitions
        WHERE workflow_id = v_workflow_id AND event_name = 'TOPIC_REJECTED'
          AND from_state_id = v_topic AND to_state_id = v_sup
    );

-- ===== STATE ACTIONS =====
-- IMPORTANT: lock team on exit from TEAM_FORMATION
INSERT INTO state_actions (state_id, name, type, trigger, config)
SELECT v_team, 'Lock team composition', 'LOCK_TEAM_COMPOSITION', 'ON_EXIT',
       '{"reason":"workflow_lock_after_team_formed"}'::jsonb
    WHERE NOT EXISTS (
        SELECT 1 FROM state_actions
        WHERE state_id = v_team AND type = 'LOCK_TEAM_COMPOSITION' AND trigger = 'ON_EXIT'
    );

RAISE NOTICE 'Demo workflow seeded. workflow_id=% department_id=%', v_workflow_id, v_dept_id;
END $$;
