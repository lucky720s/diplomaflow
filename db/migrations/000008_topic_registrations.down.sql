-- =============================================
-- Migration: 000008_topic_registrations (ROLLBACK)
-- =============================================

DROP TABLE IF EXISTS admin_topic_registration_reviews CASCADE;
DROP TABLE IF EXISTS admin_topic_registrations CASCADE;

-- Восстанавливаем letter_grade если нужен откат
ALTER TABLE admin_grades ADD COLUMN IF NOT EXISTS letter_grade VARCHAR(2);
