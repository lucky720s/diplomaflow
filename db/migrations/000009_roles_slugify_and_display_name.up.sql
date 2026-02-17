BEGIN;

-- 1) Preserve human-readable name
ALTER TABLE roles
    ADD COLUMN IF NOT EXISTS display_name VARCHAR(255);

-- Fill display_name for existing rows (if not set)
UPDATE roles
SET display_name = name
WHERE display_name IS NULL;

-- 2) Convert known seeded names to technical slugs (name becomes slug)
UPDATE roles SET name = 'professor'
WHERE name = 'Professor';

UPDATE roles SET name = 'senior_lecturer'
WHERE name = 'Senior Lecturer';

UPDATE roles SET name = 'associate_professor'
WHERE name = 'Associate Professor';

UPDATE roles SET name = 'supervisor'
WHERE name = 'Supervisor';

UPDATE roles SET name = 'norm_control'
WHERE name = 'Norm Control Reviewer';

UPDATE roles SET name = 'economics'
WHERE name = 'Economics Reviewer';

UPDATE roles SET name = 'commission'
WHERE name = 'Commission Member';

COMMIT;
