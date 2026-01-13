-- Исправляем перепутанные type/trigger в demo seed
UPDATE state_actions
SET type = 'SEND_NOTIFICATION',
    trigger = 'ON_ENTER',
    updated_at = NOW()
WHERE name = 'Notify: Select Supervisor'
  AND (type = 'ON_ENTER' AND trigger = 'SEND_NOTIFICATION');
