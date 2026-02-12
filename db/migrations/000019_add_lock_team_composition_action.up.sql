BEGIN;

-- Добавляем LOCK_TEAM_COMPOSITION на ON_EXIT для всех TEAM_FORMATION шагов,
-- если такого action ещё нет.
INSERT INTO state_actions (
    state_id,
    name,
    type,
    trigger,
    order_index,
    config,
    is_enabled,
    is_optional,
    conditions,
    max_retries,
    retry_delay_seconds,
    created_at,
    updated_at
)
SELECT
    s.id,
    'Lock team composition',
    'LOCK_TEAM_COMPOSITION',
    'ON_EXIT',
    1000,
    '{"reason":"workflow_lock_after_team_formed"}'::jsonb,
    true,
    false,
    '[]'::jsonb,
    3,
    60,
    NOW(),
    NOW()
FROM states s
WHERE s.type = 'TEAM_FORMATION'
  AND s.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1
    FROM state_actions a
    WHERE a.state_id = s.id
      AND a.type = 'LOCK_TEAM_COMPOSITION'
      AND a.trigger = 'ON_EXIT'
);

COMMIT;
