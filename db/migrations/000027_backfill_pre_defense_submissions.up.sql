-- Backfill pre-defense submissions for teams that already uploaded
-- pre-defense materials before the auto-create fix.
--
-- Причина: раньше авто-создание admin_pre_defense_submissions срабатывало
-- только на точных именах этапов (PRE_DEFENSE_1_UPLOAD/PRE_DEFENSE_2_UPLOAD/
-- PRE_DEFENSE), но реальные имена этапов в workflow PI — PRE_DEFENSE_1 и
-- PRE_DEFENSE_2 (миграция 000005). Из-за этого по уже сданным материалам
-- запись предзащиты не создавалась, и комиссия не могла выставить оценку
-- ("Предзащита ещё не создана в системе").
--
-- Эта миграция создаёт недостающие записи (status='scheduled') для каждой
-- pending workflow-сдачи на этапе загрузки материалов предзащиты, у которой
-- ещё нет активной (pending/scheduled) записи предзащиты.

BEGIN;

INSERT INTO admin_pre_defense_submissions (
    id, team_id, project_id, submitted_by,
    status, scheduled_date, scheduled_time, location,
    duration_minutes, submitted_at, created_at, updated_at
)
SELECT
    gen_random_uuid()::text,
    sub.team_id,
    sub.project_id,
    sub.submitted_by,
    'scheduled',
    NOW(),
    'auto',
    'Not specified',
    30,
    NOW(),
    NOW(),
    NOW()
FROM admin_submissions sub
JOIN states st ON st.id = sub.state_id
WHERE sub.status = 'pending'
  AND sub.deleted_at IS NULL
  AND sub.team_id IS NOT NULL
  AND (
        UPPER(COALESCE(st.name, ''))         LIKE '%PRE_DEFENSE%'
     OR UPPER(COALESCE(st.display_name, '')) LIKE '%ПРЕДЗАЩИТ%'
      )
  AND UPPER(COALESCE(st.name, '')) NOT LIKE '%REVIEW%'
  AND NOT EXISTS (
        SELECT 1
        FROM admin_pre_defense_submissions pd
        WHERE pd.project_id = sub.project_id
          AND pd.team_id = sub.team_id
          AND pd.status IN ('pending', 'scheduled')
  );

-- История для прослеживаемости созданных задним числом записей.
INSERT INTO admin_pre_defense_history (
    submission_id, action, actor_id, old_value, new_value, comment, created_at
)
SELECT
    pd.id,
    'scheduled',
    pd.submitted_by,
    '',
    'scheduled',
    'Backfilled by migration 000027 from existing pending workflow submission',
    NOW()
FROM admin_pre_defense_submissions pd
WHERE pd.scheduled_time = 'auto'
  AND pd.location = 'Not specified'
  AND NOT EXISTS (
        SELECT 1 FROM admin_pre_defense_history h
        WHERE h.submission_id = pd.id
  );

COMMIT;
