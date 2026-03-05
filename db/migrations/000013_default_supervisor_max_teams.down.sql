BEGIN;
UPDATE workflows
SET settings = settings - 'default_supervisor_max_teams'
WHERE settings ? 'default_supervisor_max_teams';
COMMIT;
