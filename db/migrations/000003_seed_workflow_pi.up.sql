BEGIN;

DO $$
DECLARE
v_iitu_id BIGINT;
  v_pi_id BIGINT;

  v_workflow_id BIGINT;

  st_team BIGINT;
  st_sup BIGINT;
  st_topic_form BIGINT;
  st_topic_approval BIGINT;
  st_predef1 BIGINT;
  st_predef2 BIGINT;
  st_norm BIGINT;
  st_econ BIGINT;
  st_antiplag BIGINT;
  st_defense BIGINT;
  st_completed BIGINT;

  v_academic_year VARCHAR(20) := '2025-2026';
BEGIN
SELECT id INTO v_iitu_id FROM universities WHERE short_name = 'IITU' LIMIT 1;
SELECT id INTO v_pi_id FROM departments WHERE university_id = v_iitu_id AND name = 'Программная Инженерия' LIMIT 1;

IF v_pi_id IS NULL THEN
    RAISE EXCEPTION 'PI department not found';
END IF;

  -- Workflow only for PI
INSERT INTO workflows (name, description, department_id, version, is_active, settings)
VALUES (
           'Дипломный проект (ПИ)',
           'Workflow дипломного проекта для кафедры Программная Инженерия (IITU)',
           v_pi_id,
           1,
           TRUE,
           '{"team_required": true, "min_team_size": 1, "max_team_size": 4, "allow_solo_project": true, "academic_year":"2025-2026"}'::jsonb
       )
    ON CONFLICT (name, department_id, version) DO NOTHING;

SELECT id INTO v_workflow_id
FROM workflows
WHERE name = 'Дипломный проект (ПИ)' AND department_id = v_pi_id AND version = 1
    LIMIT 1;

IF v_workflow_id IS NULL THEN
    RAISE EXCEPTION 'Workflow not created';
END IF;

  -- Department config (active) for PI
INSERT INTO department_workflow_configs (department_id, workflow_id, academic_year, is_active, team_settings)
VALUES (
           v_pi_id,
           v_workflow_id,
           v_academic_year,
           TRUE,
           '{
             "allow_solo": true,
             "min_size": 1,
             "max_size": 4,
             "require_same_group": false,
             "invite_expire_days": 3
           }'::jsonb
       )
    ON CONFLICT (department_id, academic_year) DO UPDATE
                                                      SET workflow_id = EXCLUDED.workflow_id,
                                                      is_active = TRUE,
                                                      team_settings = EXCLUDED.team_settings,
                                                      updated_at = NOW();

-- ======================
-- States (linear flow)
-- ======================

INSERT INTO states (workflow_id, name, display_name, description, order_index, type, is_initial, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'TEAM_FORMATION',
           'Создание/вступление в команду',
           'Создать команду или вступить по коду',
           1,
           'TEAM_FORMATION',
           TRUE,
           14,
           '{
             "team_config": {"min_size":1,"max_size":4,"allow_solo":true,"require_leader":true}
           }'::jsonb,
           '#3B82F6',
           'users'
       ) ON CONFLICT DO NOTHING;
SELECT id INTO st_team FROM states WHERE workflow_id = v_workflow_id AND name = 'TEAM_FORMATION' LIMIT 1;

INSERT INTO states (workflow_id, name, display_name, description, order_index, type, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'SUPERVISOR_SELECTION',
           'Выбор научного руководителя',
           'Команда отправляет запрос преподавателю или преподаватель может взять команду',
           2,
           'SUPERVISOR_SELECTION',
           7,
           '{
             "supervisor_config": {
               "allowed_base_roles": ["teacher"],
               "allow_teacher_claim_without_request": true
             }
           }'::jsonb,
           '#8B5CF6',
           'user-check'
       ) ON CONFLICT DO NOTHING;
SELECT id INTO st_sup FROM states WHERE workflow_id = v_workflow_id AND name = 'SUPERVISOR_SELECTION' LIMIT 1;

INSERT INTO states (workflow_id, name, display_name, description, order_index, type, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'TOPIC_SUBMISSION',
           'Заявка на утверждение темы (RU/KZ/EN)',
           'Заполнить тему на трёх языках и отправить',
           3,
           'FORM_SUBMIT',
           7,
           '{
             "form_config": {
               "form_id": "topic_trilingual_v1"
             }
           }'::jsonb,
           '#06B6D4',
           'file-text'
       ) ON CONFLICT DO NOTHING;
SELECT id INTO st_topic_form FROM states WHERE workflow_id = v_workflow_id AND name = 'TOPIC_SUBMISSION' LIMIT 1;

INSERT INTO states (workflow_id, name, display_name, description, order_index, type, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'TOPIC_APPROVAL',
           'Утверждение темы',
           'Проверка/утверждение темы',
           4,
           'APPROVAL',
           5,
           '{
             "review_config": {"reviewer_roles":["commission","teacher"],"min_reviewers":1,"require_comment":false}
           }'::jsonb,
           '#F59E0B',
           'check-circle'
       ) ON CONFLICT DO NOTHING;
SELECT id INTO st_topic_approval FROM states WHERE workflow_id = v_workflow_id AND name = 'TOPIC_APPROVAL' LIMIT 1;

INSERT INTO states (workflow_id, name, display_name, description, order_index, type, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'PRE_DEFENSE_1_MATERIALS',
           'Материалы для предзащиты 1',
           'Загрузка материалов для предзащиты 1',
           5,
           'DOCUMENT_UPLOAD',
           14,
           '{
             "file_config":{"max_files":10,"max_size_bytes":52428800,"allowed_extensions":["pdf","doc","docx","ppt","pptx"]}
           }'::jsonb,
           '#10B981',
           'upload'
       ) ON CONFLICT DO NOTHING;
SELECT id INTO st_predef1 FROM states WHERE workflow_id = v_workflow_id AND name = 'PRE_DEFENSE_1_MATERIALS' LIMIT 1;

INSERT INTO states (workflow_id, name, display_name, description, order_index, type, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'PRE_DEFENSE_2_MATERIALS',
           'Материалы для предзащиты 2',
           'Загрузка материалов для предзащиты 2',
           6,
           'DOCUMENT_UPLOAD',
           14,
           '{
             "file_config":{"max_files":10,"max_size_bytes":52428800,"allowed_extensions":["pdf","doc","docx","ppt","pptx"]}
           }'::jsonb,
           '#10B981',
           'upload'
       ) ON CONFLICT DO NOTHING;
SELECT id INTO st_predef2 FROM states WHERE workflow_id = v_workflow_id AND name = 'PRE_DEFENSE_2_MATERIALS' LIMIT 1;

INSERT INTO states (workflow_id, name, display_name, description, order_index, type, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'NORM_CONTROL_UPLOAD',
           'Нормоконтроль',
           'Загрузка документов на нормоконтроль',
           7,
           'DOCUMENT_UPLOAD',
           10,
           '{
             "file_config":{"max_files":5,"max_size_bytes":52428800,"allowed_extensions":["pdf","doc","docx"]},
             "review_config":{"reviewer_roles":["norm_control"],"min_reviewers":1,"require_comment":false}
           }'::jsonb,
           '#A855F7',
           'shield'
       ) ON CONFLICT DO NOTHING;
SELECT id INTO st_norm FROM states WHERE workflow_id = v_workflow_id AND name = 'NORM_CONTROL_UPLOAD' LIMIT 1;

INSERT INTO states (workflow_id, name, display_name, description, order_index, type, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'ECONOMICS_UPLOAD',
           'Экономическая часть',
           'Загрузка экономической части',
           8,
           'DOCUMENT_UPLOAD',
           10,
           '{
             "file_config":{"max_files":5,"max_size_bytes":52428800,"allowed_extensions":["pdf","doc","docx","xlsx"]},
             "review_config":{"reviewer_roles":["economics"],"min_reviewers":1,"require_comment":false}
           }'::jsonb,
           '#22C55E',
           'banknote'
       ) ON CONFLICT DO NOTHING;
SELECT id INTO st_econ FROM states WHERE workflow_id = v_workflow_id AND name = 'ECONOMICS_UPLOAD' LIMIT 1;

INSERT INTO states (workflow_id, name, display_name, description, order_index, type, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'ANTIPLAGIAT_UPLOAD',
           'Антиплагиат (загрузка)',
           'Загрузка файла для проверки (интеграции пока нет)',
           9,
           'DOCUMENT_UPLOAD',
           7,
           '{
             "file_config":{"max_files":2,"max_size_bytes":52428800,"allowed_extensions":["pdf","doc","docx"]}
           }'::jsonb,
           '#EF4444',
           'search'
       ) ON CONFLICT DO NOTHING;
SELECT id INTO st_antiplag FROM states WHERE workflow_id = v_workflow_id AND name = 'ANTIPLAGIAT_UPLOAD' LIMIT 1;

INSERT INTO states (workflow_id, name, display_name, description, order_index, type, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'DEFENSE',
           'Защита диплома',
           'Финальная защита',
           10,
           'DEFENSE',
           0,
           '{}'::jsonb,
           '#0EA5E9',
           'graduation-cap'
       ) ON CONFLICT DO NOTHING;
SELECT id INTO st_defense FROM states WHERE workflow_id = v_workflow_id AND name = 'DEFENSE' LIMIT 1;

INSERT INTO states (workflow_id, name, display_name, description, order_index, type, is_final, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'COMPLETED',
           'Завершено',
           'Проект завершён',
           11,
           'COMPLETED',
           TRUE,
           0,
           '{}'::jsonb,
           '#16A34A',
           'check'
       ) ON CONFLICT DO NOTHING;
SELECT id INTO st_completed FROM states WHERE workflow_id = v_workflow_id AND name = 'COMPLETED' LIMIT 1;

-- ======================
-- Transitions (linear + reject topic)
-- ======================

INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color)
SELECT v_workflow_id, 'TEAM_FORMED', 'Команда сформирована', st_team, st_sup, 'Команда готова', 'success'
    WHERE NOT EXISTS (
    SELECT 1 FROM transitions WHERE workflow_id=v_workflow_id AND event_name='TEAM_FORMED' AND from_state_id=st_team
  );

INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color)
SELECT v_workflow_id, 'SUPERVISOR_SELECTED', 'Руководитель выбран', st_sup, st_topic_form, 'Продолжить', 'primary'
    WHERE NOT EXISTS (
    SELECT 1 FROM transitions WHERE workflow_id=v_workflow_id AND event_name='SUPERVISOR_SELECTED' AND from_state_id=st_sup
  );

INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color)
SELECT v_workflow_id, 'TOPIC_SUBMITTED', 'Тема отправлена', st_topic_form, st_topic_approval, 'Отправить', 'primary'
    WHERE NOT EXISTS (
    SELECT 1 FROM transitions WHERE workflow_id=v_workflow_id AND event_name='TOPIC_SUBMITTED' AND from_state_id=st_topic_form
  );

INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color)
SELECT v_workflow_id, 'TOPIC_APPROVED', 'Тема утверждена', st_topic_approval, st_predef1, 'Далее', 'success'
    WHERE NOT EXISTS (
    SELECT 1 FROM transitions WHERE workflow_id=v_workflow_id AND event_name='TOPIC_APPROVED' AND from_state_id=st_topic_approval
  );

INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color, confirm_text)
SELECT v_workflow_id, 'TOPIC_REJECTED', 'Тема отклонена', st_topic_approval, st_topic_form, 'Отклонить', 'danger', 'Вернуть тему на доработку?'
    WHERE NOT EXISTS (
    SELECT 1 FROM transitions WHERE workflow_id=v_workflow_id AND event_name='TOPIC_REJECTED' AND from_state_id=st_topic_approval
  );

INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color)
SELECT v_workflow_id, 'PREDEFENSE1_SUBMITTED', 'Предзащита 1 — загружено', st_predef1, st_predef2, 'Далее', 'primary'
    WHERE NOT EXISTS (
    SELECT 1 FROM transitions WHERE workflow_id=v_workflow_id AND event_name='PREDEFENSE1_SUBMITTED' AND from_state_id=st_predef1
  );

INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color)
SELECT v_workflow_id, 'PREDEFENSE2_SUBMITTED', 'Предзащита 2 — загружено', st_predef2, st_norm, 'Далее', 'primary'
    WHERE NOT EXISTS (
    SELECT 1 FROM transitions WHERE workflow_id=v_workflow_id AND event_name='PREDEFENSE2_SUBMITTED' AND from_state_id=st_predef2
  );

INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color)
SELECT v_workflow_id, 'NORMCONTROL_SUBMITTED', 'Нормоконтроль — отправлено', st_norm, st_econ, 'Далее', 'primary'
    WHERE NOT EXISTS (
    SELECT 1 FROM transitions WHERE workflow_id=v_workflow_id AND event_name='NORMCONTROL_SUBMITTED' AND from_state_id=st_norm
  );

INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color)
SELECT v_workflow_id, 'ECONOMICS_SUBMITTED', 'Экономика — отправлено', st_econ, st_antiplag, 'Далее', 'primary'
    WHERE NOT EXISTS (
    SELECT 1 FROM transitions WHERE workflow_id=v_workflow_id AND event_name='ECONOMICS_SUBMITTED' AND from_state_id=st_econ
  );

INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color)
SELECT v_workflow_id, 'ANTIPLAGIAT_SUBMITTED', 'Антиплагиат — загружено', st_antiplag, st_defense, 'Далее', 'primary'
    WHERE NOT EXISTS (
    SELECT 1 FROM transitions WHERE workflow_id=v_workflow_id AND event_name='ANTIPLAGIAT_SUBMITTED' AND from_state_id=st_antiplag
  );

INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color)
SELECT v_workflow_id, 'DEFENSE_PASSED', 'Защита пройдена', st_defense, st_completed, 'Завершить', 'success'
    WHERE NOT EXISTS (
    SELECT 1 FROM transitions WHERE workflow_id=v_workflow_id AND event_name='DEFENSE_PASSED' AND from_state_id=st_defense
  );

-- ======================
-- State action: lock team composition on exit from TEAM_FORMATION
-- ======================
INSERT INTO state_actions (state_id, name, type, trigger, order_index, config, is_enabled, conditions, max_retries, retry_delay_seconds)
SELECT
    st_team,
    'Lock team composition',
    'LOCK_TEAM_COMPOSITION',
    'ON_EXIT',
    1000,
    '{"reason":"team_formed"}'::jsonb,
    TRUE,
    '[]'::jsonb,
    3,
    60
    WHERE NOT EXISTS (
    SELECT 1 FROM state_actions WHERE state_id=st_team AND type='LOCK_TEAM_COMPOSITION' AND trigger='ON_EXIT'
  );

END $$;

COMMIT;
