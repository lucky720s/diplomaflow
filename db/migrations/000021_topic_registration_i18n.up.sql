BEGIN;

-- Официальная регистрация темы хранится на 3 языках (kz/ru/en).
-- Существующие миграции не трогаем — добавляем колонки и переносим данные здесь.
-- Заявка к руководителю (admin_supervisor_requests) остаётся с одним
-- proposed_topic и в этой миграции не меняется.

ALTER TABLE admin_topic_registrations
    ADD COLUMN IF NOT EXISTS proposed_topic_kz VARCHAR(500),
    ADD COLUMN IF NOT EXISTS proposed_topic_ru VARCHAR(500),
    ADD COLUMN IF NOT EXISTS proposed_topic_en VARCHAR(500);

-- Бэкфилл из старого единственного поля: дублируем во все три языка.
UPDATE admin_topic_registrations
SET proposed_topic_kz = COALESCE(NULLIF(proposed_topic_kz, ''), proposed_topic),
    proposed_topic_ru = COALESCE(NULLIF(proposed_topic_ru, ''), proposed_topic),
    proposed_topic_en = COALESCE(NULLIF(proposed_topic_en, ''), proposed_topic)
WHERE proposed_topic IS NOT NULL;

-- Подстраховка от NULL перед установкой NOT NULL.
UPDATE admin_topic_registrations SET proposed_topic_kz = '' WHERE proposed_topic_kz IS NULL;
UPDATE admin_topic_registrations SET proposed_topic_ru = '' WHERE proposed_topic_ru IS NULL;
UPDATE admin_topic_registrations SET proposed_topic_en = '' WHERE proposed_topic_en IS NULL;

ALTER TABLE admin_topic_registrations
    ALTER COLUMN proposed_topic_kz SET NOT NULL,
    ALTER COLUMN proposed_topic_ru SET NOT NULL,
    ALTER COLUMN proposed_topic_en SET NOT NULL;

-- Старое поле больше не нужно (официальная тема теперь i18n).
-- Оно было NOT NULL без дефолта — иначе INSERT'ы новой модели падали бы.
ALTER TABLE admin_topic_registrations DROP COLUMN IF EXISTS proposed_topic;

COMMIT;
