BEGIN;

-- Device tokens for mobile push (FCM). Один пользователь может иметь несколько
-- устройств; токен уникален глобально (один токен принадлежит одному устройству).
CREATE TABLE IF NOT EXISTS device_tokens (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       VARCHAR(512) NOT NULL,
    platform    VARCHAR(20) NOT NULL DEFAULT 'android',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_device_tokens_token UNIQUE (token)
);

CREATE INDEX IF NOT EXISTS idx_device_tokens_user_id ON device_tokens(user_id);

COMMIT;
