-- Repair: восстанавливаем недостающие переходы линейной части workflow ПОСЛЕ
-- предзащиты. В части БД секция transitions из сидов прогналась не полностью —
-- состояния существуют, а переходы между ними отсутствуют. Из-за этого approve
-- нормоконтроля падал с "transition not found ... event_name=NORMCONTROL_SUBMITTED".
--
-- Граф этапов зависит от конфигурации: ECONOMICS может отсутствовать. Поэтому
-- связываем устойчиво по фактически существующим состояниям:
--   NORM_CONTROL → (ECONOMICS если есть, иначе ANTIPLAGIAT)
--   ECONOMICS    → ANTIPLAGIAT            (только если ECONOMICS есть)
--   ANTIPLAGIAT  → DEFENSE
--   DEFENSE      → COMPLETED
-- Идемпотентно: вставляем только то, чего ещё нет (по workflow_id + event_name).

BEGIN;

DO $$
DECLARE
    w RECORD;
    st_norm     BIGINT;
    st_econ     BIGINT;
    st_antiplag BIGINT;
    st_defense  BIGINT;
    st_completed BIGINT;
    v_norm_next BIGINT;
BEGIN
    FOR w IN SELECT id FROM workflows LOOP
        SELECT id INTO st_norm      FROM states WHERE workflow_id = w.id AND name = 'NORM_CONTROL' AND deleted_at IS NULL LIMIT 1;
        SELECT id INTO st_econ      FROM states WHERE workflow_id = w.id AND name = 'ECONOMICS'    AND deleted_at IS NULL LIMIT 1;
        SELECT id INTO st_antiplag  FROM states WHERE workflow_id = w.id AND name = 'ANTIPLAGIAT'  AND deleted_at IS NULL LIMIT 1;
        SELECT id INTO st_defense   FROM states WHERE workflow_id = w.id AND name = 'DEFENSE'      AND deleted_at IS NULL LIMIT 1;
        SELECT id INTO st_completed FROM states WHERE workflow_id = w.id AND name = 'COMPLETED'    AND deleted_at IS NULL LIMIT 1;

        -- NORM_CONTROL → ECONOMICS (если есть) иначе → ANTIPLAGIAT
        v_norm_next := COALESCE(st_econ, st_antiplag);
        IF st_norm IS NOT NULL AND v_norm_next IS NOT NULL THEN
            INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color)
            SELECT w.id, 'NORMCONTROL_SUBMITTED', 'Нормоконтроль — отправлено', st_norm, v_norm_next, 'Далее', 'primary'
            WHERE NOT EXISTS (SELECT 1 FROM transitions WHERE workflow_id = w.id AND event_name = 'NORMCONTROL_SUBMITTED');
        END IF;

        -- ECONOMICS → ANTIPLAGIAT (только если экономика есть)
        IF st_econ IS NOT NULL AND st_antiplag IS NOT NULL THEN
            INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color)
            SELECT w.id, 'ECONOMICS_SUBMITTED', 'Экономика — отправлено', st_econ, st_antiplag, 'Далее', 'primary'
            WHERE NOT EXISTS (SELECT 1 FROM transitions WHERE workflow_id = w.id AND event_name = 'ECONOMICS_SUBMITTED');
        END IF;

        -- ANTIPLAGIAT → DEFENSE
        IF st_antiplag IS NOT NULL AND st_defense IS NOT NULL THEN
            INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color)
            SELECT w.id, 'ANTIPLAGIAT_SUBMITTED', 'Антиплагиат — отправлено', st_antiplag, st_defense, 'Далее', 'primary'
            WHERE NOT EXISTS (SELECT 1 FROM transitions WHERE workflow_id = w.id AND event_name = 'ANTIPLAGIAT_SUBMITTED');
        END IF;

        -- DEFENSE → COMPLETED (событие из сидов — DEFENSE_PASSED)
        IF st_defense IS NOT NULL AND st_completed IS NOT NULL THEN
            INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color)
            SELECT w.id, 'DEFENSE_PASSED', 'Защита пройдена', st_defense, st_completed, 'Завершить', 'success'
            WHERE NOT EXISTS (SELECT 1 FROM transitions WHERE workflow_id = w.id AND event_name = 'DEFENSE_PASSED');
        END IF;
    END LOOP;
END $$;

COMMIT;
