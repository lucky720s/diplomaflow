-- Модель PreDefenseDocument (internal/admin/model_pre_defense.go) содержит поле
-- CreatedAt, которое GORM мапит на колонку created_at. В исходной схеме
-- (000001) таблица admin_pre_defense_documents такой колонки не имела — был
-- только uploaded_at. Из-за этого INSERT документа предзащиты падал с ошибкой
-- "column created_at of relation admin_pre_defense_documents does not exist".
--
-- Добавляем недостающую колонку, приводя схему в соответствие с моделью.

BEGIN;

ALTER TABLE admin_pre_defense_documents
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Для уже существующих строк подставляем uploaded_at как разумное значение.
UPDATE admin_pre_defense_documents
SET created_at = uploaded_at
WHERE uploaded_at IS NOT NULL;

COMMIT;
