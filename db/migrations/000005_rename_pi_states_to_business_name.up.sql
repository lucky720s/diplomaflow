BEGIN;

-- rename states by name (only within PI workflow)
UPDATE states
SET name = 'PRE_DEFENSE_1'
WHERE name = 'PRE_DEFENSE_1_MATERIALS';

UPDATE states
SET name = 'PRE_DEFENSE_2'
WHERE name = 'PRE_DEFENSE_2_MATERIALS';

UPDATE states
SET name = 'NORM_CONTROL'
WHERE name = 'NORM_CONTROL_UPLOAD';

UPDATE states
SET name = 'ECONOMICS'
WHERE name = 'ECONOMICS_UPLOAD';

UPDATE states
SET name = 'ANTIPLAGIAT'
WHERE name = 'ANTIPLAGIAT_UPLOAD';

-- sync existing projects that reference these state_ids
UPDATE projects p
SET current_state_name = s.name
    FROM states s
WHERE p.current_state_id = s.id;

COMMIT;
