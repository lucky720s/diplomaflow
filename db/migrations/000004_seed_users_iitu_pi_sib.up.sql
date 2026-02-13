BEGIN;

DO $$
DECLARE
v_iitu_id BIGINT;
  v_pi_id BIGINT;
  v_sib_id BIGINT;
  i INT;

  v_role_supervisor BIGINT;
  v_role_norm BIGINT;
  v_role_econ BIGINT;
  v_role_comm BIGINT;
BEGIN
SELECT id INTO v_iitu_id FROM universities WHERE short_name = 'IITU' LIMIT 1;
SELECT id INTO v_pi_id FROM departments WHERE university_id = v_iitu_id AND name = 'Программная Инженерия' LIMIT 1;
SELECT id INTO v_sib_id FROM departments WHERE university_id = v_iitu_id AND name = 'СИБ' LIMIT 1;

IF v_iitu_id IS NULL OR v_pi_id IS NULL OR v_sib_id IS NULL THEN
    RAISE EXCEPTION 'IITU/PI/SIB not found';
END IF;

  -- Admin user (useful)
INSERT INTO users (email, password, first_name, last_name, role, university_id, department_id)
VALUES (
           'admin@iitu.edu.kz',
           crypt('12345678', gen_salt('bf', 10)),
           'Admin',
           'IITU',
           'admin',
           v_iitu_id,
           v_pi_id
       )
    ON CONFLICT (email) DO NOTHING;

-- Teachers (all in PI)
FOR i IN 1..20 LOOP
    INSERT INTO users (email, password, first_name, last_name, role, university_id, department_id)
    VALUES (
      format('teacher%s@iitu.edu.kz', i),
      crypt('12345678', gen_salt('bf', 10)),
      'Teacher',
      format('%s', i),
      'teacher',
      v_iitu_id,
      v_pi_id
    )
    ON CONFLICT (email) DO NOTHING;
END LOOP;

  -- Students: 1..40 -> PI, 41..50 -> SIB
FOR i IN 1..50 LOOP
    INSERT INTO users (email, password, first_name, last_name, role, university_id, department_id)
    VALUES (
      format('student%s@iitu.edu.kz', i),
      crypt('12345678', gen_salt('bf', 10)),
      'Student',
      format('%s', i),
      'student',
      v_iitu_id,
      CASE WHEN i <= 40 THEN v_pi_id ELSE v_sib_id END
    )
    ON CONFLICT (email) DO NOTHING;
END LOOP;

  -- Resolve dynamic role IDs (directory roles)
SELECT id INTO v_role_supervisor FROM roles WHERE department_id = v_pi_id AND name = 'Supervisor' LIMIT 1;
SELECT id INTO v_role_norm FROM roles WHERE department_id = v_pi_id AND name = 'Norm Control Reviewer' LIMIT 1;
SELECT id INTO v_role_econ FROM roles WHERE department_id = v_pi_id AND name = 'Economics Reviewer' LIMIT 1;
SELECT id INTO v_role_comm FROM roles WHERE department_id = v_pi_id AND name = 'Commission Member' LIMIT 1;

-- Assign dynamic roles to teachers (examples)
-- teacher1..teacher10 => Supervisor
FOR i IN 1..10 LOOP
    INSERT INTO user_role_assignments (user_id, role_id, department_id, assigned_by, comment)
SELECT u.id, v_role_supervisor, v_pi_id, (SELECT id FROM users WHERE email='admin@iitu.edu.kz' LIMIT 1), 'seed'
FROM users u
WHERE u.email = format('teacher%s@iitu.edu.kz', i)
  AND v_role_supervisor IS NOT NULL
ON CONFLICT DO NOTHING;
END LOOP;

  -- teacher11..teacher15 => Norm Control Reviewer
FOR i IN 11..15 LOOP
    INSERT INTO user_role_assignments (user_id, role_id, department_id, assigned_by, comment)
SELECT u.id, v_role_norm, v_pi_id, (SELECT id FROM users WHERE email='admin@iitu.edu.kz' LIMIT 1), 'seed'
FROM users u
WHERE u.email = format('teacher%s@iitu.edu.kz', i)
  AND v_role_norm IS NOT NULL
ON CONFLICT DO NOTHING;
END LOOP;

  -- teacher16..teacher18 => Economics Reviewer
FOR i IN 16..18 LOOP
    INSERT INTO user_role_assignments (user_id, role_id, department_id, assigned_by, comment)
SELECT u.id, v_role_econ, v_pi_id, (SELECT id FROM users WHERE email='admin@iitu.edu.kz' LIMIT 1), 'seed'
FROM users u
WHERE u.email = format('teacher%s@iitu.edu.kz', i)
  AND v_role_econ IS NOT NULL
ON CONFLICT DO NOTHING;
END LOOP;

  -- teacher19..teacher20 => Commission Member
FOR i IN 19..20 LOOP
    INSERT INTO user_role_assignments (user_id, role_id, department_id, assigned_by, comment)
SELECT u.id, v_role_comm, v_pi_id, (SELECT id FROM users WHERE email='admin@iitu.edu.kz' LIMIT 1), 'seed'
FROM users u
WHERE u.email = format('teacher%s@iitu.edu.kz', i)
  AND v_role_comm IS NOT NULL
ON CONFLICT DO NOTHING;
END LOOP;

END $$;

COMMIT;
