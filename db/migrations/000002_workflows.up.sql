-- =============================================
-- Migration: 000002_workflows
-- Description: Workflow tables with extended fields
-- =============================================

-- Workflows (расширенная версия)
CREATE TABLE IF NOT EXISTS workflows (
                                         id BIGSERIAL PRIMARY KEY,
                                         name VARCHAR(255) NOT NULL,
    description TEXT,
    department_id BIGINT NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    version INT DEFAULT 1,
    is_active BOOLEAN DEFAULT FALSE,
    is_template BOOLEAN DEFAULT FALSE,
    parent_id BIGINT REFERENCES workflows(id) ON DELETE SET NULL,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
                                                                 UNIQUE(name, department_id, version)
    );

CREATE INDEX IF NOT EXISTS idx_workflows_department_id ON workflows(department_id);
CREATE INDEX IF NOT EXISTS idx_workflows_is_active ON workflows(department_id, is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_workflows_is_template ON workflows(is_template) WHERE is_template = true;
CREATE INDEX IF NOT EXISTS idx_workflows_parent_id ON workflows(parent_id);
CREATE INDEX IF NOT EXISTS idx_workflows_deleted_at ON workflows(deleted_at);

-- States (расширенная версия)
CREATE TABLE IF NOT EXISTS states (
                                      id BIGSERIAL PRIMARY KEY,
                                      workflow_id BIGINT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    order_index INT NOT NULL DEFAULT 0,
    type VARCHAR(50) NOT NULL,

    -- Флаги состояния
    is_initial BOOLEAN DEFAULT FALSE,
    is_final BOOLEAN DEFAULT FALSE,
    is_optional BOOLEAN DEFAULT FALSE,

    -- Конфигурация
    config JSONB NOT NULL DEFAULT '{}',

    -- Временные настройки
    duration_days INT DEFAULT 0,
    duration_mode VARCHAR(20) DEFAULT 'relative',
    fixed_deadline TIMESTAMP WITH TIME ZONE,

                                                                                               -- UI настройки
                                                                                               color VARCHAR(20),
    icon VARCHAR(50),

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
                                                                                               );

CREATE INDEX IF NOT EXISTS idx_states_workflow_id ON states(workflow_id);
CREATE INDEX IF NOT EXISTS idx_states_order ON states(workflow_id, order_index);
CREATE INDEX IF NOT EXISTS idx_states_type ON states(type);
CREATE INDEX IF NOT EXISTS idx_states_is_initial ON states(workflow_id, is_initial) WHERE is_initial = true;
CREATE INDEX IF NOT EXISTS idx_states_deleted_at ON states(deleted_at);

-- Transitions (расширенная версия)
CREATE TABLE IF NOT EXISTS transitions (
                                           id BIGSERIAL PRIMARY KEY,
                                           workflow_id BIGINT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    event_name VARCHAR(100) NOT NULL,
    display_name VARCHAR(255),
    from_state_id BIGINT NOT NULL REFERENCES states(id) ON DELETE CASCADE,
    to_state_id BIGINT NOT NULL REFERENCES states(id) ON DELETE CASCADE,

    -- Условия перехода
    conditions JSONB DEFAULT '[]',

    -- UI
    button_label VARCHAR(100),
    button_color VARCHAR(20) DEFAULT 'primary',
    confirm_text TEXT,

    -- Приоритет (для множественных переходов)
    priority INT DEFAULT 0,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(workflow_id, event_name, from_state_id)
    );

CREATE INDEX IF NOT EXISTS idx_transitions_from_state ON transitions(from_state_id);
CREATE INDEX IF NOT EXISTS idx_transitions_to_state ON transitions(to_state_id);
CREATE INDEX IF NOT EXISTS idx_transitions_workflow_id ON transitions(workflow_id);
CREATE INDEX IF NOT EXISTS idx_transitions_event ON transitions(from_state_id, event_name);

-- State Actions (расширенная версия)
CREATE TABLE IF NOT EXISTS state_actions (
                                             id BIGSERIAL PRIMARY KEY,
                                             state_id BIGINT NOT NULL REFERENCES states(id) ON DELETE CASCADE,
    name VARCHAR(100),
    type VARCHAR(50) NOT NULL,
    trigger VARCHAR(50) NOT NULL,
    order_index INT DEFAULT 0,
    config JSONB NOT NULL DEFAULT '{}',
    is_enabled BOOLEAN DEFAULT TRUE,
    conditions JSONB DEFAULT '[]',

    -- Retry settings
    max_retries INT DEFAULT 3,
    retry_delay_seconds INT DEFAULT 60,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_state_actions_state_id ON state_actions(state_id);
CREATE INDEX IF NOT EXISTS idx_state_actions_trigger ON state_actions(state_id, trigger);
CREATE INDEX IF NOT EXISTS idx_state_actions_enabled ON state_actions(state_id, is_enabled) WHERE is_enabled = true;

-- State Conditions (для сложных правил)
CREATE TABLE IF NOT EXISTS state_conditions (
                                                id BIGSERIAL PRIMARY KEY,
                                                state_id BIGINT NOT NULL REFERENCES states(id) ON DELETE CASCADE,
    name VARCHAR(100),
    type VARCHAR(50) NOT NULL,
    expression TEXT,
    config JSONB DEFAULT '{}',
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_state_conditions_state ON state_conditions(state_id);

-- Workflow Templates (для быстрого создания)
CREATE TABLE IF NOT EXISTS workflow_templates (
                                                  id BIGSERIAL PRIMARY KEY,
                                                  name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    category VARCHAR(100), -- 'diploma', 'coursework', 'practice'
    template_data JSONB NOT NULL, -- полная структура workflow
    is_system BOOLEAN DEFAULT FALSE, -- системный шаблон
    created_by BIGINT REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_workflow_templates_category ON workflow_templates(category);
