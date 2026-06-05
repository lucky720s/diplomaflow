-- Откат бэкфилла нормоконтроля. Удаляем только нетронутые авто-созданные
-- записи (status='submitted', без назначенного проверяющего, без начала
-- проверки/issues/истории) — реальные/проверенные записи не трогаем.

BEGIN;

DELETE FROM norm_control_checks nc
WHERE nc.status = 'submitted'
  AND nc.checker_id IS NULL
  AND nc.started_at IS NULL
  AND nc.reviewed_at IS NULL
  AND NOT EXISTS (
        SELECT 1 FROM norm_control_issues i WHERE i.submission_id = nc.submission_id
  )
  AND NOT EXISTS (
        SELECT 1 FROM norm_control_history h WHERE h.submission_id = nc.submission_id
  );

COMMIT;
