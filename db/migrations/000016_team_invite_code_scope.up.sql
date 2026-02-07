-- 1) Добавляем колонки (сначала nullable, чтобы спокойно backfill-нуть)
ALTER TABLE teams
    ADD COLUMN IF NOT EXISTS university_id BIGINT,
    ADD COLUMN IF NOT EXISTS department_id BIGINT,
    ADD COLUMN IF NOT EXISTS invite_code VARCHAR(6),
    ADD COLUMN IF NOT EXISTS composition_locked BOOLEAN DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS composition_locked_at TIMESTAMPTZ;

-- 2) Backfill university_id/department_id по лидеру команды
UPDATE teams t
SET
    university_id = u.university_id,
    department_id = u.department_id
    FROM team_members tm
JOIN users u ON u.id = tm.user_id
WHERE tm.team_id = t.id
  AND tm.role = 'leader'
  AND (t.university_id IS NULL OR t.department_id IS NULL);

-- 3) Если вдруг лидер не найден, добиваем любым участником
UPDATE teams t
SET
    university_id = u.university_id,
    department_id = u.department_id
    FROM team_members tm
JOIN users u ON u.id = tm.user_id
WHERE tm.team_id = t.id
  AND (t.university_id IS NULL OR t.department_id IS NULL);

-- 4) Сгенерим invite_code тем командам, у кого его нет (простая генерация в SQL)
-- (дальше в коде будет нормальная крипто-генерация)
-- используем base36-кусок от md5, но строго 6 символов
UPDATE teams
SET invite_code = SUBSTRING(UPPER(MD5(RANDOM()::text)) FOR 6)
WHERE invite_code IS NULL OR invite_code = '';

-- 5) Делаем NOT NULL
ALTER TABLE teams
    ALTER COLUMN university_id SET NOT NULL,
ALTER COLUMN department_id SET NOT NULL,
  ALTER COLUMN invite_code SET NOT NULL,
  ALTER COLUMN composition_locked SET NOT NULL;

-- 6) Индексы/уникальность:
-- уникальность кода ТОЛЬКО в рамках университета
CREATE UNIQUE INDEX IF NOT EXISTS ux_teams_university_invite_code
    ON teams(university_id, invite_code);

CREATE INDEX IF NOT EXISTS idx_teams_department_id ON teams(department_id);
