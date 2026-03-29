ALTER TABLE users ADD COLUMN is_active BOOLEAN DEFAULT TRUE;

-- если хочешь убрать soft delete полностью:
ALTER TABLE users DROP COLUMN deleted_at;
