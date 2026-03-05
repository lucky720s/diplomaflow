BEGIN;
UPDATE workflows
SET settings = settings || '{"default_supervisor_max_teams": 5}'::jsonb
WHERE settings IS NOT NULL
  AND NOT (settings ? 'default_supervisor_max_teams');
COMMIT;
