BEGIN;

DO $$
DECLARE
  v_iitu_id   BIGINT;
  v_pi_id     BIGINT;
  v_workflow_id BIGINT;

  st_topic_approval BIGINT;
  st_predef1_mat    BIGINT;
  st_predef1_review BIGINT;
  st_predef2_mat    BIGINT;
  st_predef2_review BIGINT;
  st_antiplag       BIGINT;
  st_defense        BIGINT;
  st_completed      BIGINT;
  st_norm           BIGINT;
  st_econ           BIGINT;
BEGIN
  SELECT id INTO v_iitu_id FROM universities WHERE short_name = 'IITU' LIMIT 1;
  SELECT id INTO v_pi_id   FROM departments WHERE university_id = v_iitu_id AND name = 'Компьютерная инженерия' LIMIT 1;
  SELECT id INTO v_workflow_id FROM workflows WHERE department_id = v_pi_id AND is_active = TRUE LIMIT 1;

  IF v_workflow_id IS NULL THEN
    RAISE EXCEPTION 'Active workflow for PI not found';
  END IF;

  -- Получаем ID стэйтов
  SELECT id INTO st_topic_approval FROM states WHERE workflow_id = v_workflow_id AND name = 'TOPIC_APPROVAL'         LIMIT 1;
  SELECT id INTO st_predef1_mat    FROM states WHERE workflow_id = v_workflow_id AND name = 'PRE_DEFENSE_1_MATERIALS' LIMIT 1;
  SELECT id INTO st_predef2_mat    FROM states WHERE workflow_id = v_workflow_id AND name = 'PRE_DEFENSE_2_MATERIALS' LIMIT 1;
  SELECT id INTO st_antiplag       FROM states WHERE workflow_id = v_workflow_id AND name = 'ANTIPLAGIAT_UPLOAD'      LIMIT 1;
  SELECT id INTO st_defense        FROM states WHERE workflow_id = v_workflow_id AND name = 'DEFENSE'                 LIMIT 1;
  SELECT id INTO st_completed      FROM states WHERE workflow_id = v_workflow_id AND name = 'COMPLETED'               LIMIT 1;
  SELECT id INTO st_norm           FROM states WHERE workflow_id = v_workflow_id AND name = 'NORM_CONTROL_UPLOAD'     LIMIT 1;
  SELECT id INTO st_econ           FROM states WHERE workflow_id = v_workflow_id AND name = 'ECONOMICS_UPLOAD'        LIMIT 1;

  -- ================================================================
  -- 1. Создаём стэйты-ревью для предзащит (отдельные стэйты оценки)
  -- ================================================================

  -- PRE_DEFENSE_1_REVIEW
  INSERT INTO states (workflow_id, name, display_name, description, order_index, type, duration_days, config, color, icon)
  VALUES (
    v_workflow_id,
    'PRE_DEFENSE_1_REVIEW',
    'Оценка предзащиты 1',
    'Комиссия выставляет оценки за предзащиту 1',
    5,  -- между PRE_DEFENSE_1_MATERIALS (5) и PRE_DEFENSE_2_MATERIALS (6)
    'REVIEW',
    14,
    '{
      "review_config": {
        "reviewer_roles": ["commission"],
        "finalizer_role": "dean_office",
        "grade_type": "score",
        "passing_score": 50,
        "min_reviewers": 0,
        "require_comment": false
      }
    }'::jsonb,
    '#F97316',
    'star'
  ) ON CONFLICT DO NOTHING;
  SELECT id INTO st_predef1_review FROM states WHERE workflow_id = v_workflow_id AND name = 'PRE_DEFENSE_1_REVIEW' LIMIT 1;

  -- Сдвигаем order_index стэйтов после 5 вверх
  UPDATE states SET order_index = order_index + 1
  WHERE workflow_id = v_workflow_id AND order_index >= 6 AND name != 'PRE_DEFENSE_1_REVIEW';

  -- PRE_DEFENSE_2_REVIEW
  INSERT INTO states (workflow_id, name, display_name, description, order_index, type, duration_days, config, color, icon)
  VALUES (
    v_workflow_id,
    'PRE_DEFENSE_2_REVIEW',
    'Оценка предзащиты 2',
    'Комиссия выставляет оценки за предзащиту 2',
    8,
    'REVIEW',
    14,
    '{
      "review_config": {
        "reviewer_roles": ["commission"],
        "finalizer_role": "dean_office",
        "grade_type": "score",
        "passing_score": 50,
        "min_reviewers": 0,
        "require_comment": false
      }
    }'::jsonb,
    '#F97316',
    'star'
  ) ON CONFLICT DO NOTHING;
  SELECT id INTO st_predef2_review FROM states WHERE workflow_id = v_workflow_id AND name = 'PRE_DEFENSE_2_REVIEW' LIMIT 1;

  -- Сдвигаем order_index стэйтов после 8 вверх
  UPDATE states SET order_index = order_index + 1
  WHERE workflow_id = v_workflow_id AND order_index >= 9 AND name NOT IN ('PRE_DEFENSE_1_REVIEW', 'PRE_DEFENSE_2_REVIEW');

  -- ================================================================
  -- 2. Обновляем review_config у существующих стэйтов
  -- ================================================================

  -- TOPIC_APPROVAL → admission, finalizer = dean_office
  UPDATE states SET config = jsonb_set(
    config,
    '{review_config}',
    '{
      "reviewer_roles": ["commission", "teacher"],
      "finalizer_role": "dean_office",
      "grade_type": "admission",
      "passing_score": 0,
      "min_reviewers": 1,
      "require_comment": false
    }'::jsonb
  )
  WHERE id = st_topic_approval;

  -- ANTIPLAGIAT_UPLOAD → admission, finalizer = dean_office
  UPDATE states SET config = jsonb_set(
    COALESCE(config, '{}'::jsonb),
    '{review_config}',
    '{
      "reviewer_roles": ["norm_control"],
      "finalizer_role": "dean_office",
      "grade_type": "admission",
      "passing_score": 0,
      "min_reviewers": 1,
      "require_comment": false
    }'::jsonb
  )
  WHERE id = st_antiplag;

  -- DEFENSE → score, finalizer = dean_office
  UPDATE states SET config = jsonb_set(
    COALESCE(config, '{}'::jsonb),
    '{review_config}',
    '{
      "reviewer_roles": ["commission"],
      "finalizer_role": "dean_office",
      "grade_type": "score",
      "passing_score": 50,
      "min_reviewers": 0,
      "require_comment": false
    }'::jsonb
  )
  WHERE id = st_defense;

  -- ================================================================
  -- 3. Перестраиваем transitions для review-стэйтов
  -- ================================================================

  -- PRE_DEFENSE_1_MATERIALS → PRE_DEFENSE_1_REVIEW (вместо прямого перехода в predef2)
  UPDATE transitions
  SET to_state_id = st_predef1_review
  WHERE workflow_id = v_workflow_id AND event_name = 'PREDEFENSE1_SUBMITTED' AND from_state_id = st_predef1_mat;

  -- PRE_DEFENSE_1_REVIEW → PRE_DEFENSE_2_MATERIALS (pass)
  INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color, confirm_text,
    conditions)
  SELECT v_workflow_id, 'PREDEFENSE1_PASSED', 'Предзащита 1 — допущена', st_predef1_review, st_predef2_mat, 'Допустить', 'success',
    'Перевести на предзащиту 2?',
    '[
      {"type":"role","operator":"in","value":["dean_office"]},
      {"type":"field","field":"predefense1_review_all_voted","operator":"eq","value":true},
      {"type":"field","field":"predefense1_review_passed","operator":"eq","value":true}
    ]'::jsonb
  WHERE NOT EXISTS (
    SELECT 1 FROM transitions WHERE workflow_id = v_workflow_id AND event_name = 'PREDEFENSE1_PASSED'
  );

  -- PRE_DEFENSE_1_REVIEW → PRE_DEFENSE_1_MATERIALS (fail)
  INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color, confirm_text,
    conditions)
  SELECT v_workflow_id, 'PREDEFENSE1_FAILED', 'Предзащита 1 — не допущена', st_predef1_review, st_predef1_mat, 'На доработку', 'danger',
    'Вернуть на доработку материалы предзащиты 1?',
    '[
      {"type":"role","operator":"in","value":["dean_office"]},
      {"type":"field","field":"predefense1_review_all_voted","operator":"eq","value":true},
      {"type":"field","field":"predefense1_review_failed","operator":"eq","value":true}
    ]'::jsonb
  WHERE NOT EXISTS (
    SELECT 1 FROM transitions WHERE workflow_id = v_workflow_id AND event_name = 'PREDEFENSE1_FAILED'
  );

  -- PRE_DEFENSE_2_MATERIALS → PRE_DEFENSE_2_REVIEW (вместо NORM_CONTROL)
  UPDATE transitions
  SET to_state_id = st_predef2_review
  WHERE workflow_id = v_workflow_id AND event_name = 'PREDEFENSE2_SUBMITTED' AND from_state_id = st_predef2_mat;

  -- PRE_DEFENSE_2_REVIEW → NORM_CONTROL_UPLOAD (pass)
  INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color, confirm_text,
    conditions)
  SELECT v_workflow_id, 'PREDEFENSE2_PASSED', 'Предзащита 2 — допущена', st_predef2_review, st_norm, 'Допустить', 'success',
    'Перевести на нормоконтроль?',
    '[
      {"type":"role","operator":"in","value":["dean_office"]},
      {"type":"field","field":"predefense2_review_all_voted","operator":"eq","value":true},
      {"type":"field","field":"predefense2_review_passed","operator":"eq","value":true}
    ]'::jsonb
  WHERE NOT EXISTS (
    SELECT 1 FROM transitions WHERE workflow_id = v_workflow_id AND event_name = 'PREDEFENSE2_PASSED'
  );

  -- PRE_DEFENSE_2_REVIEW → PRE_DEFENSE_2_MATERIALS (fail)
  INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color, confirm_text,
    conditions)
  SELECT v_workflow_id, 'PREDEFENSE2_FAILED', 'Предзащита 2 — не допущена', st_predef2_review, st_predef2_mat, 'На доработку', 'danger',
    'Вернуть на доработку материалы предзащиты 2?',
    '[
      {"type":"role","operator":"in","value":["dean_office"]},
      {"type":"field","field":"predefense2_review_all_voted","operator":"eq","value":true},
      {"type":"field","field":"predefense2_review_failed","operator":"eq","value":true}
    ]'::jsonb
  WHERE NOT EXISTS (
    SELECT 1 FROM transitions WHERE workflow_id = v_workflow_id AND event_name = 'PREDEFENSE2_FAILED'
  );

  -- TOPIC_APPROVAL: обновляем conditions у существующих переходов
  UPDATE transitions SET conditions = '[
    {"type":"role","operator":"in","value":["dean_office"]},
    {"type":"field","field":"topic_approval_all_voted","operator":"eq","value":true},
    {"type":"field","field":"topic_approval_passed","operator":"eq","value":true}
  ]'::jsonb
  WHERE workflow_id = v_workflow_id AND event_name = 'TOPIC_APPROVED' AND from_state_id = st_topic_approval;

  UPDATE transitions SET conditions = '[
    {"type":"role","operator":"in","value":["dean_office"]},
    {"type":"field","field":"topic_approval_all_voted","operator":"eq","value":true},
    {"type":"field","field":"topic_approval_failed","operator":"eq","value":true}
  ]'::jsonb
  WHERE workflow_id = v_workflow_id AND event_name = 'TOPIC_REJECTED' AND from_state_id = st_topic_approval;

  -- ANTIPLAGIAT → DEFENSE: обновляем conditions
  UPDATE transitions SET conditions = '[
    {"type":"role","operator":"in","value":["dean_office"]},
    {"type":"field","field":"antiplagiat_review_all_voted","operator":"eq","value":true},
    {"type":"field","field":"antiplagiat_review_passed","operator":"eq","value":true}
  ]'::jsonb
  WHERE workflow_id = v_workflow_id AND event_name = 'ANTIPLAGIAT_SUBMITTED' AND from_state_id = st_antiplag;

  -- ANTIPLAGIAT → ANTIPLAGIAT (fail — вернуть на загрузку)
  INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color, confirm_text,
    conditions)
  SELECT v_workflow_id, 'ANTIPLAGIAT_REJECTED', 'Антиплагиат — отклонён', st_antiplag, st_antiplag, 'Отклонить', 'danger',
    'Вернуть на повторную загрузку?',
    '[
      {"type":"role","operator":"in","value":["dean_office"]},
      {"type":"field","field":"antiplagiat_review_all_voted","operator":"eq","value":true},
      {"type":"field","field":"antiplagiat_review_failed","operator":"eq","value":true}
    ]'::jsonb
  WHERE NOT EXISTS (
    SELECT 1 FROM transitions WHERE workflow_id = v_workflow_id AND event_name = 'ANTIPLAGIAT_REJECTED'
  );

  -- DEFENSE transitions
  UPDATE transitions SET conditions = '[
    {"type":"role","operator":"in","value":["dean_office"]},
    {"type":"field","field":"defense_review_all_voted","operator":"eq","value":true},
    {"type":"field","field":"defense_review_passed","operator":"eq","value":true}
  ]'::jsonb
  WHERE workflow_id = v_workflow_id AND event_name = 'DEFENSE_PASSED' AND from_state_id = st_defense;

  -- DEFENSE FAILED → COMPLETED (сохраняем проект с результатом, но не пересдачу — решается на уровне бизнеса)
  INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color, confirm_text,
    conditions)
  SELECT v_workflow_id, 'DEFENSE_FAILED', 'Защита не пройдена', st_defense, st_completed, 'Завершить (не зачёт)', 'danger',
    'Завершить проект с неудовлетворительной оценкой?',
    '[
      {"type":"role","operator":"in","value":["dean_office"]},
      {"type":"field","field":"defense_review_all_voted","operator":"eq","value":true},
      {"type":"field","field":"defense_review_failed","operator":"eq","value":true}
    ]'::jsonb
  WHERE NOT EXISTS (
    SELECT 1 FROM transitions WHERE workflow_id = v_workflow_id AND event_name = 'DEFENSE_FAILED'
  );

  -- ================================================================
  -- 4. ON_EXIT действия REVIEW_GATE для review-стэйтов
  -- ================================================================

  -- TOPIC_APPROVAL
  INSERT INTO state_actions (state_id, name, type, trigger, order_index, config, is_enabled, conditions, max_retries, retry_delay_seconds)
  SELECT st_topic_approval, 'Review Gate — Тема', 'REVIEW_GATE', 'ON_EXIT', 100,
    '{"patch_key":"topic_approval"}'::jsonb, TRUE, '[]'::jsonb, 1, 0
  WHERE NOT EXISTS (
    SELECT 1 FROM state_actions WHERE state_id = st_topic_approval AND type = 'REVIEW_GATE' AND trigger = 'ON_EXIT'
  );

  -- ANTIPLAGIAT_UPLOAD
  INSERT INTO state_actions (state_id, name, type, trigger, order_index, config, is_enabled, conditions, max_retries, retry_delay_seconds)
  SELECT st_antiplag, 'Review Gate — Антиплагиат', 'REVIEW_GATE', 'ON_EXIT', 100,
    '{"patch_key":"antiplagiat_review"}'::jsonb, TRUE, '[]'::jsonb, 1, 0
  WHERE NOT EXISTS (
    SELECT 1 FROM state_actions WHERE state_id = st_antiplag AND type = 'REVIEW_GATE' AND trigger = 'ON_EXIT'
  );

  -- PRE_DEFENSE_1_REVIEW
  INSERT INTO state_actions (state_id, name, type, trigger, order_index, config, is_enabled, conditions, max_retries, retry_delay_seconds)
  SELECT st_predef1_review, 'Review Gate — Предзащита 1', 'REVIEW_GATE', 'ON_EXIT', 100,
    '{"patch_key":"predefense1_review"}'::jsonb, TRUE, '[]'::jsonb, 1, 0
  WHERE NOT EXISTS (
    SELECT 1 FROM state_actions WHERE state_id = st_predef1_review AND type = 'REVIEW_GATE' AND trigger = 'ON_EXIT'
  );

  -- PRE_DEFENSE_2_REVIEW
  INSERT INTO state_actions (state_id, name, type, trigger, order_index, config, is_enabled, conditions, max_retries, retry_delay_seconds)
  SELECT st_predef2_review, 'Review Gate — Предзащита 2', 'REVIEW_GATE', 'ON_EXIT', 100,
    '{"patch_key":"predefense2_review"}'::jsonb, TRUE, '[]'::jsonb, 1, 0
  WHERE NOT EXISTS (
    SELECT 1 FROM state_actions WHERE state_id = st_predef2_review AND type = 'REVIEW_GATE' AND trigger = 'ON_EXIT'
  );

  -- DEFENSE
  INSERT INTO state_actions (state_id, name, type, trigger, order_index, config, is_enabled, conditions, max_retries, retry_delay_seconds)
  SELECT st_defense, 'Review Gate — Защита', 'REVIEW_GATE', 'ON_EXIT', 100,
    '{"patch_key":"defense_review"}'::jsonb, TRUE, '[]'::jsonb, 1, 0
  WHERE NOT EXISTS (
    SELECT 1 FROM state_actions WHERE state_id = st_defense AND type = 'REVIEW_GATE' AND trigger = 'ON_EXIT'
  );

END $$;

COMMIT;
