BEGIN;

CREATE TABLE IF NOT EXISTS antiplag_checks (
    submission_id    VARCHAR(36) PRIMARY KEY REFERENCES admin_submissions(id) ON DELETE CASCADE,

    project_id       BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    team_id          BIGINT REFERENCES teams(id) ON DELETE SET NULL,
    step_id          BIGINT REFERENCES states(id) ON DELETE SET NULL,

    primary_file_id  VARCHAR(36) REFERENCES file_metadata(id) ON DELETE SET NULL,
    file_ids         JSONB NOT NULL DEFAULT '[]'::jsonb,

    document_version INT NOT NULL DEFAULT 1,

    checker_id       BIGINT REFERENCES users(id) ON DELETE SET NULL,

    plagiarism_score INT,
    ai_score         INT,

    status           VARCHAR(50) NOT NULL DEFAULT 'submitted',
    overall_comment  TEXT,

    started_at       TIMESTAMPTZ,
    reviewed_at      TIMESTAMPTZ,

    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_antiplag_plagiarism CHECK (plagiarism_score IS NULL OR (plagiarism_score BETWEEN 0 AND 100)),
    CONSTRAINT chk_antiplag_ai         CHECK (ai_score IS NULL OR (ai_score BETWEEN 0 AND 100))
    );

CREATE INDEX IF NOT EXISTS idx_antiplag_checks_project ON antiplag_checks(project_id);
CREATE INDEX IF NOT EXISTS idx_antiplag_checks_status  ON antiplag_checks(status);
CREATE INDEX IF NOT EXISTS idx_antiplag_checks_checker ON antiplag_checks(checker_id);
CREATE INDEX IF NOT EXISTS idx_antiplag_checks_step    ON antiplag_checks(step_id);

CREATE TABLE IF NOT EXISTS antiplag_comments (
    id            UUID PRIMARY KEY,
    submission_id VARCHAR(36) NOT NULL REFERENCES antiplag_checks(submission_id) ON DELETE CASCADE,

    text          TEXT NOT NULL,
    page_number   INT,

    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_antiplag_comments_submission ON antiplag_comments(submission_id);

CREATE TABLE IF NOT EXISTS antiplag_history (
    id            BIGSERIAL PRIMARY KEY,
    project_id    BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    submission_id VARCHAR(36) REFERENCES admin_submissions(id) ON DELETE SET NULL,

    action        VARCHAR(100) NOT NULL,
    actor_id      BIGINT REFERENCES users(id) ON DELETE SET NULL,
    comment       TEXT,

    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_antiplag_history_project    ON antiplag_history(project_id);
CREATE INDEX IF NOT EXISTS idx_antiplag_history_submission ON antiplag_history(submission_id);

DO $$
    DECLARE
        v_iitu_id BIGINT;
        v_pi_id   BIGINT;
    BEGIN
    SELECT id INTO v_iitu_id FROM universities WHERE short_name = 'IITU' LIMIT 1;
    IF v_iitu_id IS NULL THEN
        RAISE NOTICE 'IITU university not found, skip antiplagiat role seed';
        RETURN;
    END IF;

    SELECT id INTO v_pi_id FROM departments
    WHERE university_id = v_iitu_id AND name = 'Компьютерная инженерия' LIMIT 1;
    IF v_pi_id IS NULL THEN
            RAISE NOTICE 'PI department not found, skip antiplagiat role seed';
            RETURN;
    END IF;

INSERT INTO roles (name, display_name, department_id)
VALUES ('antiplagiat', 'Antiplagiat Reviewer', v_pi_id)
    ON CONFLICT (name, department_id) DO NOTHING;
END $$;

UPDATE states
SET config = jsonb_set(
        COALESCE(config, '{}'::jsonb),
        '{review_config}',
        '{
          "reviewer_roles": ["antiplagiat"],
          "finalizer_role": "dean_office",
          "grade_type": "admission",
          "passing_score": 0,
          "min_reviewers": 1,
          "require_comment": false
        }'::jsonb
             )
WHERE name = 'ANTIPLAGIAT';

UPDATE state_actions sa
SET is_enabled = FALSE
    FROM states s
WHERE sa.state_id = s.id
  AND s.name = 'ANTIPLAGIAT'
  AND sa.type IN ('CHECK_ANTIPLAGIAT', 'EXTERNAL')
  AND COALESCE(sa.config->>'plugin_id', sa.name) ILIKE '%ANTIPLAGIAT%'
  AND sa.type <> 'REVIEW_GATE';

UPDATE action_registry
SET is_enabled = FALSE
WHERE id = 'CHECK_ANTIPLAGIAT';

COMMIT;
