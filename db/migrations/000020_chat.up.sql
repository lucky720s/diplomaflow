BEGIN;

-- Беседы (диалоги/группы).
CREATE TABLE IF NOT EXISTS conversations (
    id          BIGSERIAL PRIMARY KEY,
    type        VARCHAR(20) NOT NULL DEFAULT 'direct', -- direct | group
    title       VARCHAR(255) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Участники беседы. last_read_message_id — водяной знак прочитанного для unread-счётчика.
CREATE TABLE IF NOT EXISTS conversation_participants (
    id                   BIGSERIAL PRIMARY KEY,
    conversation_id      BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id              BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_read_message_id BIGINT NOT NULL DEFAULT 0,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_conversation_participant UNIQUE (conversation_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_participants_user ON conversation_participants(user_id);

-- Сообщения.
CREATE TABLE IF NOT EXISTS chat_messages (
    id              BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content         TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_chat_messages_conv ON chat_messages(conversation_id, id DESC);

COMMIT;
