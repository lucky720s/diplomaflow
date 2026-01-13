-- =============================================
-- Migration: 000006_seed_demo_workflow
-- Description: Demo workflow for CS department
-- =============================================

-- Создаём workflow (без хардкода ID!)
INSERT INTO workflows (name, description, department_id, version, is_active, settings)
SELECT
    'Дипломный проект 2025',
    'Стандартный процесс дипломной работы',
    d.id,
    1,
    true,
    '{
        "team_required": true,
        "min_team_size": 1,
        "max_team_size": 3,
        "allow_solo_project": true,
        "academic_year": "2024-2025",
        "deadline_warning_days": [7, 3, 1],
        "allowed_file_types": [".pdf", ".doc", ".docx"]
    }'::jsonb
FROM departments d
WHERE d.name = 'Computer Science'
  AND d.university_id = (SELECT id FROM universities WHERE short_name = 'IITU')
    ON CONFLICT (name, department_id, version) DO NOTHING;

-- Создаём States (используем переменные, не хардкод ID)
DO $$
DECLARE
v_workflow_id BIGINT;
    v_state_team_formation BIGINT;
    v_state_supervisor BIGINT;
    v_state_topic_approval BIGINT;
    v_state_task_upload BIGINT;
    v_state_work_progress BIGINT;
    v_state_document_submission BIGINT;
    v_state_plagiarism_check BIGINT;
    v_state_supervisor_review BIGINT;
    v_state_defense BIGINT;
    v_state_grading BIGINT;
    v_state_completed BIGINT;
BEGIN
    -- Получаем ID workflow
SELECT id INTO v_workflow_id
FROM workflows
WHERE name = 'Дипломный проект 2025'
  AND department_id = (
    SELECT d.id FROM departments d
                         JOIN universities u ON d.university_id = u.id
    WHERE d.name = 'Computer Science' AND u.short_name = 'IITU'
)
    LIMIT 1;

IF v_workflow_id IS NULL THEN
        RAISE EXCEPTION 'Workflow not found!';
END IF;

    -- State 1: Team Formation
INSERT INTO states (workflow_id, name, display_name, description, order_index, type, is_initial, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'TEAM_FORMATION',
           'Формирование команды',
           'Создайте команду или начните индивидуальный проект',
           1,
           'TEAM_FORMATION',
           true,
           14,
           '{
               "team_config": {
                   "min_size": 1,
                   "max_size": 3,
                   "allow_solo": true,
                   "require_leader": true,
                   "invite_expire_days": 3
               }
           }'::jsonb,
           '#3B82F6',
           'users'
       )
    RETURNING id INTO v_state_team_formation;

-- State 2: Supervisor Selection
INSERT INTO states (workflow_id, name, display_name, description, order_index, type, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'SUPERVISOR_SELECTION',
           'Выбор руководителя',
           'Выберите научного руководителя и согласуйте тему',
           2,
           'SUPERVISOR_SELECTION',
           7,
           '{
               "supervisor_config": {
                   "allowed_roles": ["teacher", "professor"],
                   "max_students_per_supervisor": 5,
                   "require_topic_proposal": true
               }
           }'::jsonb,
           '#8B5CF6',
           'user-check'
       )
    RETURNING id INTO v_state_supervisor;

-- State 3: Topic Approval
INSERT INTO states (workflow_id, name, display_name, description, order_index, type, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'TOPIC_APPROVAL',
           'Утверждение темы',
           'Тема должна быть утверждена кафедрой',
           3,
           'APPROVAL',
           5,
           '{
               "review_config": {
                   "reviewer_roles": ["teacher", "head_of_department"],
                   "require_comment": false,
                   "min_reviewers": 1
               }
           }'::jsonb,
           '#F59E0B',
           'check-circle'
       )
    RETURNING id INTO v_state_topic_approval;

-- State 4: Task Upload
INSERT INTO states (workflow_id, name, display_name, description, order_index, type, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'TASK_UPLOAD',
           'Загрузка задания',
           'Загрузите подписанное задание на дипломную работу',
           4,
           'DOCUMENT_UPLOAD',
           7,
           '{
               "file_config": {
                   "max_files": 1,
                   "max_size_bytes": 10485760,
                   "allowed_extensions": [".pdf"],
                   "required_files": [
                       {
                           "type": "task_sheet",
                           "display_name": "Задание на дипломную работу",
                           "extensions": [".pdf"],
                           "template_url": "/templates/task_sheet.pdf"
                       }
                   ]
               }
           }'::jsonb,
           '#10B981',
           'upload'
       )
    RETURNING id INTO v_state_task_upload;

-- State 5: Work in Progress
INSERT INTO states (workflow_id, name, display_name, description, order_index, type, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'WORK_IN_PROGRESS',
           'Работа над проектом',
           'Выполняйте работу согласно плану',
           5,
           'MILESTONE',
           90,
           '{
               "milestones": [
                   {"name": "Теоретическая часть", "deadline_offset_days": 30, "weight": 0.2},
                   {"name": "Практическая реализация", "deadline_offset_days": 60, "weight": 0.4},
                   {"name": "Тестирование", "deadline_offset_days": 80, "weight": 0.2},
                   {"name": "Документация", "deadline_offset_days": 90, "weight": 0.2}
               ]
           }'::jsonb,
           '#6366F1',
           'code'
       )
    RETURNING id INTO v_state_work_progress;

-- State 6: Document Submission
INSERT INTO states (workflow_id, name, display_name, description, order_index, type, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'DOCUMENT_SUBMISSION',
           'Сдача работы',
           'Загрузите финальную версию дипломной работы',
           6,
           'DOCUMENT_UPLOAD',
           14,
           '{
               "file_config": {
                   "max_files": 5,
                   "max_size_bytes": 52428800,
                   "required_files": [
                       {"type": "diploma_text", "display_name": "Текст дипломной работы", "extensions": [".pdf", ".docx"]},
                       {"type": "presentation", "display_name": "Презентация", "extensions": [".pdf", ".pptx"]},
                       {"type": "abstract", "display_name": "Аннотация", "extensions": [".pdf"]}
                   ]
               }
           }'::jsonb,
           '#EC4899',
           'file-text'
       )
    RETURNING id INTO v_state_document_submission;

-- State 7: Plagiarism Check
INSERT INTO states (workflow_id, name, display_name, description, order_index, type, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'PLAGIARISM_CHECK',
           'Проверка на антиплагиат',
           'Автоматическая проверка на оригинальность',
           7,
           'EXTERNAL_CHECK',
           3,
           '{
               "external_check_config": {
                   "service_type": "antiplagiat",
                   "min_score": 70,
                   "auto_reject": false
               }
           }'::jsonb,
           '#EF4444',
           'shield-check'
       )
    RETURNING id INTO v_state_plagiarism_check;

-- State 8: Supervisor Review
INSERT INTO states (workflow_id, name, display_name, description, order_index, type, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'SUPERVISOR_REVIEW',
           'Отзыв руководителя',
           'Руководитель пишет отзыв на работу',
           8,
           'REVIEW',
           7,
           '{
               "review_config": {
                   "reviewer_roles": ["supervisor"],
                   "require_comment": true,
                   "allow_grade": false
               }
           }'::jsonb,
           '#F97316',
           'message-square'
       )
    RETURNING id INTO v_state_supervisor_review;

-- State 9: Defense
INSERT INTO states (workflow_id, name, display_name, description, order_index, type, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'DEFENSE',
           'Защита',
           'Защита дипломной работы перед комиссией',
           9,
           'DEFENSE',
           1,
           '{
               "defense_config": {
                   "commission_roles": ["commission_member", "commission_head"],
                   "min_commission_size": 3,
                   "presentation_time_minutes": 15,
                   "questions_time_minutes": 10
               }
           }'::jsonb,
           '#7C3AED',
           'award'
       )
    RETURNING id INTO v_state_defense;

-- State 10: Grading
INSERT INTO states (workflow_id, name, display_name, description, order_index, type, duration_days, config, color, icon)
VALUES (
           v_workflow_id,
           'GRADING',
           'Оценивание',
           'Выставление итоговой оценки',
           10,
           'GRADING',
           1,
           '{
               "grading_config": {
                   "grade_scale": "letter",
                   "passing_score": 50,
                   "components": [
                       {"name": "Работа над проектом", "weight": 0.3, "grader_role": "supervisor"},
                       {"name": "Качество работы", "weight": 0.4, "grader_role": "commission"},
                       {"name": "Защита", "weight": 0.3, "grader_role": "commission"}
                   ],
                   "weighted_average": true
               }
           }'::jsonb,
           '#14B8A6',
           'star'
       )
    RETURNING id INTO v_state_grading;

-- State 11: Completed
INSERT INTO states (workflow_id, name, display_name, description, order_index, type, is_final, config, color, icon)
VALUES (
           v_workflow_id,
           'COMPLETED',
           'Завершено',
           'Дипломный проект успешно завершён!',
           11,
           'COMPLETED',
           true,
           '{}'::jsonb,
           '#22C55E',
           'check'
       )
    RETURNING id INTO v_state_completed;

-- =============================================
-- TRANSITIONS (теперь с переменными, не хардкодом!)
-- =============================================

-- Happy path transitions
INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color) VALUES
                                                                                                                            (v_workflow_id, 'TEAM_FORMED', 'Команда создана', v_state_team_formation, v_state_supervisor, 'Команда готова', 'success'),
                                                                                                                            (v_workflow_id, 'SUPERVISOR_SELECTED', 'Руководитель выбран', v_state_supervisor, v_state_topic_approval, 'Отправить на утверждение', 'primary'),
                                                                                                                            (v_workflow_id, 'TOPIC_APPROVED', 'Тема утверждена', v_state_topic_approval, v_state_task_upload, 'Утвердить тему', 'success'),
                                                                                                                            (v_workflow_id, 'TASK_UPLOADED', 'Задание загружено', v_state_task_upload, v_state_work_progress, 'Подтвердить загрузку', 'success'),
                                                                                                                            (v_workflow_id, 'WORK_COMPLETED', 'Работа завершена', v_state_work_progress, v_state_document_submission, 'Сдать работу', 'primary'),
                                                                                                                            (v_workflow_id, 'DOCUMENTS_SUBMITTED', 'Документы сданы', v_state_document_submission, v_state_plagiarism_check, 'Отправить на проверку', 'primary'),
                                                                                                                            (v_workflow_id, 'CHECK_PASSED', 'Проверка пройдена', v_state_plagiarism_check, v_state_supervisor_review, 'Продолжить', 'success'),
                                                                                                                            (v_workflow_id, 'REVIEW_APPROVED', 'Отзыв получен', v_state_supervisor_review, v_state_defense, 'Допустить к защите', 'success'),
                                                                                                                            (v_workflow_id, 'DEFENSE_COMPLETED', 'Защита завершена', v_state_defense, v_state_grading, 'Перейти к оцениванию', 'primary'),
                                                                                                                            (v_workflow_id, 'GRADE_SUBMITTED', 'Оценка выставлена', v_state_grading, v_state_completed, 'Завершить', 'success');

-- Rejection/revision transitions
INSERT INTO transitions (workflow_id, event_name, display_name, from_state_id, to_state_id, button_label, button_color, confirm_text) VALUES
                                                                                                                                          (v_workflow_id, 'TOPIC_REJECTED', 'Тема отклонена', v_state_topic_approval, v_state_supervisor, 'Отклонить', 'danger', 'Вы уверены, что хотите отклонить тему?'),
                                                                                                                                          (v_workflow_id, 'CHECK_FAILED', 'Проверка не пройдена', v_state_plagiarism_check, v_state_document_submission, 'На доработку', 'warning', NULL),
                                                                                                                                          (v_workflow_id, 'REVIEW_REVISION', 'Требуется доработка', v_state_supervisor_review, v_state_document_submission, 'На доработку', 'warning', 'Отправить работу на доработку?');

-- =============================================
-- STATE ACTIONS (уведомления и автоматизация)
-- =============================================

-- Уведомление при входе в Team Formation
INSERT INTO state_actions (state_id, name, type, trigger, config) VALUES
    (v_state_team_formation, 'Notify: Start Team Formation', 'SEND_NOTIFICATION', 'ON_ENTER',
     '{"title": "Начните формирование команды", "message": "У вас есть 14 дней для создания команды или начала индивидуального проекта", "link": "/teams"}'::jsonb);

-- Уведомление при выборе руководителя
INSERT INTO state_actions (state_id, name, type, trigger, config) VALUES
    (v_state_supervisor, 'Notify: Select Supervisor',  'SEND_NOTIFICATION','ON_ENTER',
     '{"title": "Выберите научного руководителя", "message": "Свяжитесь с преподавателем и согласуйте тему дипломной работы"}'::jsonb);

-- Уведомление об утверждении темы
INSERT INTO state_actions (state_id, name, type, trigger, config) VALUES
    (v_state_topic_approval, 'Notify: Topic Pending', 'SEND_NOTIFICATION', 'ON_ENTER',
     '{"title": "Тема на утверждении", "message": "Ваша тема отправлена на рассмотрение кафедрой"}'::jsonb);

-- Уведомление о deadline
INSERT INTO state_actions (state_id, name, type, trigger, config) VALUES
    (v_state_task_upload, 'Notify: Deadline Warning', 'SEND_NOTIFICATION', 'ON_DEADLINE',
     '{"title": "⚠️ Срок загрузки истекает!", "message": "Осталось менее 24 часов для загрузки задания", "urgency": "high"}'::jsonb);

-- Уведомление о завершении
INSERT INTO state_actions (state_id, name, type, trigger, config) VALUES
    (v_state_completed, 'Notify: Congratulations', 'SEND_NOTIFICATION', 'ON_ENTER',
     '{"title": "🎉 Поздравляем!", "message": "Вы успешно защитили дипломную работу!"}'::jsonb);

RAISE NOTICE 'Demo workflow created successfully!';
    RAISE NOTICE 'Workflow ID: %', v_workflow_id;
    RAISE NOTICE 'States created: 11';
    RAISE NOTICE 'Transitions created: 13';
END $$;
