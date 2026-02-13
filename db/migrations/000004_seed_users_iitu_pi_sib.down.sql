BEGIN;

-- Revoke assignments for seeded users
DELETE FROM user_role_assignments
WHERE user_id IN (
    SELECT id FROM users
    WHERE email = 'admin@iitu.edu.kz'
       OR email ~ '^teacher[0-9]+@iitu\.edu\.kz$'
   OR email ~ '^student[0-9]+@iitu\.edu\.kz$'
    );

DELETE FROM refresh_tokens
WHERE user_id IN (
    SELECT id FROM users
    WHERE email = 'admin@iitu.edu.kz'
       OR email ~ '^teacher[0-9]+@iitu\.edu\.kz$'
   OR email ~ '^student[0-9]+@iitu\.edu\.kz$'
    );

DELETE FROM users
WHERE email = 'admin@iitu.edu.kz'
   OR email ~ '^teacher[0-9]+@iitu\.edu\.kz$'
   OR email ~ '^student[0-9]+@iitu\.edu\.kz$';

COMMIT;
