-- Откат: удаляем review-стэйты и восстанавливаем прямые переходы
BEGIN;

DO $$
DECLARE
  v_iitu_id     BIGINT;
  v_pi_id       BIGINT;
  v_workflow_id BIGINT;
  st_predef1_mat    BIGINT;
  st_predef2_mat    BIGINT;
  st_predef1_review BIGINT;
  st_predef2_review BIGINT;
  st_norm           BIGINT;
BEGIN
  SELECT id INTO v_iitu_id FROM universities WHERE short_name = 'IITU' LIMIT 1;
  SELECT id INTO v_pi_id   FROM departments WHERE university_id = v_iitu_id AND name = 'Компьютерная инженерия' LIMIT 1;
  SELECT id INTO v_workflow_id FROM workflows WHERE department_id = v_pi_id AND is_active = TRUE LIMIT 1;

  SELECT id INTO st_predef1_mat    FROM states WHERE workflow_id = v_workflow_id AND name = 'PRE_DEFENSE_1_MATERIALS' LIMIT 1;
  SELECT id INTO st_predef2_mat    FROM states WHERE workflow_id = v_workflow_id AND name = 'PRE_DEFENSE_2_MATERIALS' LIMIT 1;
  SELECT id INTO st_predef1_review FROM states WHERE workflow_id = v_workflow_id AND name = 'PRE_DEFENSE_1_REVIEW'    LIMIT 1;
  SELECT id INTO st_predef2_review FROM states WHERE workflow_id = v_workflow_id AND name = 'PRE_DEFENSE_2_REVIEW'    LIMIT 1;
  SELECT id INTO st_norm           FROM states WHERE workflow_id = v_workflow_id AND name = 'NORM_CONTROL_UPLOAD'     LIMIT 1;

  -- Восстанавливаем прямые переходы
  UPDATE transitions SET to_state_id = st_predef2_mat
  WHERE workflow_id = v_workflow_id AND event_name = 'PREDEFENSE1_SUBMITTED' AND from_state_id = st_predef1_mat;

  UPDATE transitions SET to_state_id = st_norm
  WHERE workflow_id = v_workflow_id AND event_name = 'PREDEFENSE2_SUBMITTED' AND from_state_id = st_predef2_mat;

  -- Удаляем переходы review-стэйтов
  DELETE FROM transitions WHERE workflow_id = v_workflow_id AND event_name IN (
    'PREDEFENSE1_PASSED','PREDEFENSE1_FAILED','PREDEFENSE2_PASSED','PREDEFENSE2_FAILED',
    'ANTIPLAGIAT_REJECTED','DEFENSE_FAILED'
  );

  -- Удаляем review-стэйты (cascade удалит state_actions)
  DELETE FROM state_actions WHERE state_id IN (st_predef1_review, st_predef2_review);
  DELETE FROM states WHERE id IN (st_predef1_review, st_predef2_review);

  -- Убираем REVIEW_GATE из остальных стэйтов
  DELETE FROM state_actions WHERE type = 'REVIEW_GATE' AND trigger = 'ON_EXIT';

  -- Сбрасываем conditions у transitions
  UPDATE transitions SET conditions = '[]'::jsonb
  WHERE workflow_id = v_workflow_id AND event_name IN ('TOPIC_APPROVED','TOPIC_REJECTED','ANTIPLAGIAT_SUBMITTED','DEFENSE_PASSED');
END $$;

COMMIT;
