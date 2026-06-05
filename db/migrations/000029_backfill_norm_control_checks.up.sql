-- Backfill norm_control_checks for documents uploaded before the norm-control
-- auto-create fix.
--
-- Причина: раньше ветка NORM_CONTROL в SubmitDocumentForStep была пустой —
-- запись norm_control_checks не создавалась, поэтому загруженный студентом
-- файл не попадал в очередь нормоконтролёра (она читает из norm_control_checks).
--
-- Эта миграция создаёт недостающие записи (status='submitted') для каждой
-- pending workflow-сдачи на этапе нормоконтроля, у которой ещё нет проверки.
-- Идемпотентна за счёт NOT EXISTS. primary_file_id подставляется только если
-- файл реально существует в file_metadata (иначе NULL — чтобы не нарушить FK).

BEGIN;

INSERT INTO norm_control_checks (
    submission_id, project_id, team_id, step_id,
    primary_file_id, file_ids, document_version,
    status, created_at, updated_at
)
SELECT
    sub.id,
    sub.project_id,
    sub.team_id,
    sub.state_id,
    (SELECT fm.id FROM file_metadata fm WHERE fm.id = (sub.files->>0)),
    COALESCE(sub.files, '[]'::jsonb),
    1 + COALESCE((
        SELECT MAX(nc2.document_version)
        FROM norm_control_checks nc2
        WHERE nc2.project_id = sub.project_id
          AND nc2.step_id = sub.state_id
    ), 0),
    'submitted',
    NOW(),
    NOW()
FROM admin_submissions sub
JOIN states st ON st.id = sub.state_id
WHERE sub.deleted_at IS NULL
  AND sub.status = 'pending'
  AND jsonb_typeof(COALESCE(sub.files, '[]'::jsonb)) = 'array'
  AND UPPER(COALESCE(st.name, '')) IN ('NORM_CONTROL', 'NORMO_CONTROL', 'NORM_CONTROL_UPLOAD')
  AND NOT EXISTS (
        SELECT 1 FROM norm_control_checks nc
        WHERE nc.submission_id = sub.id
  );

COMMIT;
