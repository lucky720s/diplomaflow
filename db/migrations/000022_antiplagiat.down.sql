BEGIN;

UPDATE action_registry
SET is_enabled = TRUE
WHERE id = 'CHECK_ANTIPLAGIAT';

UPDATE state_actions sa
SET is_enabled = TRUE
    FROM states s
WHERE sa.state_id = s.id
  AND s.name = 'ANTIPLAGIAT'
  AND sa.type IN ('CHECK_ANTIPLAGIAT', 'EXTERNAL')
  AND COALESCE(sa.config->>'plugin_id', sa.name) ILIKE '%ANTIPLAGIAT%'
  AND sa.type <> 'REVIEW_GATE';

UPDATE states
SET config = jsonb_set(
        COALESCE(config, '{}'::jsonb),
        '{review_config}',
        '{
          "reviewer_roles": ["norm_control"],
          "finalizer_role": "dean_office",
          "grade_type": "admission",
          "passing_score": 0,
          "min_reviewers": 1,
          "require_comment": false
        }'::jsonb
             )
WHERE name = 'ANTIPLAGIAT';

DELETE FROM roles WHERE name = 'antiplagiat';

DROP TABLE IF EXISTS antiplag_history CASCADE;
DROP TABLE IF EXISTS antiplag_comments CASCADE;
DROP TABLE IF EXISTS antiplag_checks CASCADE;

COMMIT;
