-- =============================================================================
-- reset_team_progress.sql
-- ПОЛНЫЙ сброс прогресса этапов ВСЕХ команд к началу (TEAM_FORMATION).
--
-- Удаляет ВСЕ данные прогресса:
--   • сабмиты документов и их ревью          (admin_submissions, admin_submission_reviews)
--   • оценки и историю оценок                (admin_grades, admin_grade_history)
--   • предзащиты (записи/документы/комиссия/история)
--   • антиплагиат (проверки/комментарии/история)
--   • нормоконтроль (проверки/issues/история/чек-листы)
--   • историю переходов состояний            (state_histories)
--   • регистрации тем и их ревью             (admin_topic_registrations, ...reviews)
-- И возвращает каждый projects.* в начальное состояние workflow (is_initial).
--
-- НЕ трогает: пользователей, команды, кафедры, сами workflow/states/transitions,
--             доски задач, файлы (file_metadata), уведомления.
--
-- ВНИМАНИЕ: операция необратима. Сделай дамп перед запуском:
--   docker compose exec -T main_postgres pg_dump -U diplomaflow diplomaflow > backup_before_reset.sql
--
-- Запуск (локально через docker compose):
--   docker compose exec -T main_postgres psql -U diplomaflow -d diplomaflow -v ON_ERROR_STOP=1 -f - < db/scripts/reset_team_progress.sql
-- или скопировать содержимое и выполнить в psql/Beekeeper/DBeaver.
-- =============================================================================

BEGIN;

-- --- Предпросмотр: сколько строк будет затронуто ----------------------------
DO $$
DECLARE
    v_projects   bigint;
    v_subs       bigint;
    v_predef     bigint;
BEGIN
    SELECT count(*) INTO v_projects FROM projects;
    SELECT count(*) INTO v_subs     FROM admin_submissions;
    SELECT count(*) INTO v_predef   FROM admin_pre_defense_submissions;
    RAISE NOTICE 'Reset scope: projects=%, admin_submissions=%, pre_defense_submissions=%',
        v_projects, v_subs, v_predef;
END $$;

-- --- 1. Антиплагиат (дети → родитель) ---------------------------------------
DELETE FROM antiplag_comments;
DELETE FROM antiplag_history;
DELETE FROM antiplag_checks;

-- --- 2. Нормоконтроль (дети → родитель) -------------------------------------
DELETE FROM norm_control_issues;
DELETE FROM norm_control_history;
DELETE FROM norm_control_checklists;
DELETE FROM norm_control_checks;

-- --- 3. Предзащита (дети → родитель) -----------------------------------------
DELETE FROM admin_pre_defense_history;
DELETE FROM admin_pre_defense_commission;
DELETE FROM admin_pre_defense_documents;
DELETE FROM admin_pre_defense_submissions;

-- --- 4. Оценки ---------------------------------------------------------------
DELETE FROM admin_grade_history;
DELETE FROM admin_grades;

-- --- 5. Сабмиты документов и их ревью ----------------------------------------
DELETE FROM admin_submission_reviews;
DELETE FROM admin_submissions;

-- --- 6. История переходов состояний ------------------------------------------
DELETE FROM state_histories;

-- --- 7. Регистрации тем (полный сброс включает темы) -------------------------
DELETE FROM admin_topic_registration_reviews;
DELETE FROM admin_topic_registrations;

-- --- 8. Возврат проектов в начальное состояние workflow ----------------------
-- Начальное состояние = states.is_initial = true; если флаг не выставлен —
-- берём состояние с минимальным order_index в том же workflow.
WITH init AS (
    SELECT DISTINCT ON (s.workflow_id)
        s.workflow_id,
        s.id   AS state_id,
        s.name AS state_name
    FROM states s
    WHERE s.deleted_at IS NULL
    ORDER BY s.workflow_id, (s.is_initial = true) DESC, s.order_index ASC
)
UPDATE projects p
SET current_state_id   = init.state_id,
    current_state_name = init.state_name,
    status             = init.state_name,  -- движок workflow хранит status == имя текущего состояния
    data               = '{}'::jsonb,
    topic_registered_at = NULL,
    deadline_at        = NULL,
    deadline_processed = FALSE,
    updated_at         = now()
FROM init
WHERE p.workflow_id = init.workflow_id;

-- --- Контроль результата -----------------------------------------------------
DO $$
DECLARE
    v_left_subs   bigint;
    v_not_initial bigint;
BEGIN
    SELECT count(*) INTO v_left_subs FROM admin_submissions;
    SELECT count(*) INTO v_not_initial
    FROM projects p
    JOIN states s ON s.id = p.current_state_id
    WHERE COALESCE(s.is_initial, false) = false
      AND p.current_state_id IS NOT NULL;
    RAISE NOTICE 'After reset: admin_submissions left=%, projects NOT at initial state=%',
        v_left_subs, v_not_initial;
END $$;

-- Если всё ок — COMMIT. Хочешь сначала проверить — замени на ROLLBACK.
COMMIT;
