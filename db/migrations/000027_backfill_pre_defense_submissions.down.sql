-- Откат бэкфилла предзащит. Удаляем только записи, созданные миграцией 000027:
-- их можно опознать по сигнатуре авто-создания (scheduled_time='auto',
-- location='Not specified', status='scheduled', нет оценки) и по записи истории
-- с пометкой backfill. Реальные/оценённые предзащиты не трогаем.

BEGIN;

DELETE FROM admin_pre_defense_submissions pd
WHERE pd.status = 'scheduled'
  AND pd.scheduled_time = 'auto'
  AND pd.location = 'Not specified'
  AND pd.grade IS NULL
  AND pd.graded_at IS NULL
  AND EXISTS (
        SELECT 1 FROM admin_pre_defense_history h
        WHERE h.submission_id = pd.id
          AND h.comment = 'Backfilled by migration 000027 from existing pending workflow submission'
  );

COMMIT;
