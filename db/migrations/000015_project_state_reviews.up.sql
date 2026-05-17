BEGIN;

CREATE TABLE IF NOT EXISTS project_state_reviews (
    id            BIGSERIAL PRIMARY KEY,
    project_id    BIGINT       NOT NULL,
    state_id      BIGINT       NOT NULL,
    reviewer_id   BIGINT       NOT NULL,
    role_slug     VARCHAR(64)  NOT NULL,
    -- admission: 'approved' | 'rejected'
    -- score: NULL (используется поле score)
    decision      VARCHAR(16),
    -- 0-100 для score-стэйтов
    score         SMALLINT     CHECK (score IS NULL OR (score >= 0 AND score <= 100)),
    comment       TEXT         NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, state_id, reviewer_id)
);

CREATE INDEX IF NOT EXISTS idx_psr_project_state   ON project_state_reviews (project_id, state_id);
CREATE INDEX IF NOT EXISTS idx_psr_reviewer        ON project_state_reviews (reviewer_id);

COMMIT;
