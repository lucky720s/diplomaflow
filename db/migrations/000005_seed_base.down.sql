-- This is seed data, we don't typically rollback seeds
-- But if needed:
DELETE FROM users WHERE email = 'admin@diplomaflow.kz';
DELETE FROM roles WHERE department_id IN (SELECT id FROM departments WHERE university_id IN (1,2,3));
DELETE FROM departments WHERE university_id IN (1,2,3);
DELETE FROM universities WHERE short_name IN ('IITU', 'AITU', 'КазНУ');
