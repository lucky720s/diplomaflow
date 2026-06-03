BEGIN;

-- Возвращаем старое единственное поле, заполняя его русским вариантом.
ALTER TABLE admin_topic_registrations
    ADD COLUMN IF NOT EXISTS proposed_topic VARCHAR(500);

UPDATE admin_topic_registrations
SET proposed_topic = COALESCE(NULLIF(proposed_topic_ru, ''), NULLIF(proposed_topic_en, ''), proposed_topic_kz, '')
WHERE proposed_topic IS NULL;

ALTER TABLE admin_topic_registrations ALTER COLUMN proposed_topic SET NOT NULL;

ALTER TABLE admin_topic_registrations
    DROP COLUMN IF EXISTS proposed_topic_kz,
    DROP COLUMN IF EXISTS proposed_topic_ru,
    DROP COLUMN IF EXISTS proposed_topic_en;

COMMIT;
