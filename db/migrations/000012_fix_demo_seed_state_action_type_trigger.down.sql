-- Откат: возвращаем назад (если нужно)
UPDATE state_actions
SET type = 'ON_ENTER',
    trigger = 'SEND_NOTIFICATION',
    updated_at = NOW()
WHERE name = 'Notify: Select Supervisor'
  AND (type = 'SEND_NOTIFICATION' AND trigger = 'ON_ENTER');
