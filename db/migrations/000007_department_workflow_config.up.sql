-- Конфигурация workflow для каждой кафедры
CREATE TABLE department_workflow_configs (
                                             id BIGSERIAL PRIMARY KEY,
                                             department_id BIGINT NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
                                             workflow_id BIGINT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,

    -- Настройки для этой кафедры
                                             academic_year VARCHAR(20) NOT NULL,           -- "2024-2025"
                                             is_active BOOLEAN DEFAULT false,

    -- Переопределение настроек workflow
                                             config_overrides JSONB DEFAULT '{}',

    -- Настройки команд
                                             team_settings JSONB DEFAULT '{
                                               "allow_solo": true,
                                               "min_size": 1,
                                               "max_size": 3,
                                               "require_same_group": false
                                             }',

    -- Дедлайны (переопределяют duration_days в states)
                                             deadline_overrides JSONB DEFAULT '{}',

                                             created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
                                             updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

                                             UNIQUE(department_id, academic_year)
);

-- Кастомные этапы для кафедры (дополнительные к workflow)
CREATE TABLE department_custom_steps (
                                         id BIGSERIAL PRIMARY KEY,
                                         department_config_id BIGINT NOT NULL REFERENCES department_workflow_configs(id) ON DELETE CASCADE,

                                         name VARCHAR(255) NOT NULL,
                                         display_name VARCHAR(255) NOT NULL,
                                         step_type VARCHAR(50) NOT NULL,

    -- Позиция: после какого state_id вставить
                                         insert_after_state_id BIGINT REFERENCES states(id),

    -- Конфигурация этапа
                                         config JSONB NOT NULL DEFAULT '{}',

    -- Обязательность
                                         is_required BOOLEAN DEFAULT true,

    -- Длительность
                                         duration_days INT DEFAULT 7,

                                         created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Реестр доступных действий (для админки)
CREATE TABLE action_registry (
                                 id VARCHAR(50) PRIMARY KEY,              -- "SEND_NOTIFICATION", "CHECK_ANTIPLAGIAT"
                                 name VARCHAR(255) NOT NULL,
                                 description TEXT,
                                 category VARCHAR(50),                    -- "notification", "validation", "external"
                                 config_schema JSONB NOT NULL,            -- JSON Schema для конфигурации
                                 is_system BOOLEAN DEFAULT false,         -- Системное действие
                                 is_enabled BOOLEAN DEFAULT true,
                                 created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Seed базовых действий
INSERT INTO action_registry (id, name, description, category, config_schema, is_system) VALUES
                                                                                            ('SEND_NOTIFICATION', 'Отправить уведомление', 'Отправляет push-уведомление пользователю', 'notification',
                                                                                             '{"type":"object","properties":{"title":{"type":"string"},"message":{"type":"string"},"recipients":{"type":"array"}}}', true),
                                                                                            ('SEND_EMAIL', 'Отправить email', 'Отправляет email на указанный адрес', 'notification',
                                                                                             '{"type":"object","properties":{"template_id":{"type":"string"},"to":{"type":"array"}}}', true),
                                                                                            ('CHECK_ANTIPLAGIAT', 'Проверка на антиплагиат', 'Отправляет документ на проверку в систему Антиплагиат', 'external',
                                                                                             '{"type":"object","properties":{"min_score":{"type":"number"},"service_url":{"type":"string"}}}', true),
                                                                                            ('VALIDATE_FILES', 'Валидация файлов', 'Проверяет загруженные файлы на соответствие требованиям', 'validation',
                                                                                             '{"type":"object","properties":{"allowed_extensions":{"type":"array"},"max_size_mb":{"type":"number"}}}', true),
                                                                                            ('ASSIGN_REVIEWER', 'Назначить проверяющего', 'Автоматически назначает проверяющего из пула', 'workflow',
                                                                                             '{"type":"object","properties":{"reviewer_roles":{"type":"array"},"assignment_strategy":{"type":"string"}}}', true),
                                                                                            ('SCHEDULE_REMINDER', 'Запланировать напоминание', 'Создаёт отложенное напоминание', 'notification',
                                                                                             '{"type":"object","properties":{"days_before":{"type":"number"},"message":{"type":"string"}}}', true),
                                                                                            ('WEBHOOK', 'Вызвать внешний API', 'Отправляет HTTP запрос на внешний сервис', 'external',
                                                                                             '{"type":"object","properties":{"url":{"type":"string"},"method":{"type":"string"},"headers":{"type":"object"}}}', true);

CREATE INDEX idx_dept_wf_config_active ON department_workflow_configs(department_id, is_active) WHERE is_active = true;
CREATE INDEX idx_dept_wf_config_year ON department_workflow_configs(department_id, academic_year);
