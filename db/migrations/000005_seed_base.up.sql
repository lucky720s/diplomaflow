-- =============================================
-- Migration: 000005_seed_base
-- Description: Base seed data (universities, etc.)
-- =============================================

-- Universities
INSERT INTO universities (name, short_name) VALUES
                                                ('International Information Technology University', 'IITU'),
                                                ('Astana IT University', 'AITU'),
                                                ('Казахский Национальный Университет', 'КазНУ')
    ON CONFLICT (name) DO NOTHING;

-- Departments for IITU (university_id = 1)
INSERT INTO departments (name, university_id) VALUES
                                                  ('Computer Science', 1),
                                                  ('Software Engineering', 1),
                                                  ('Information Systems', 1),
                                                  ('Cybersecurity', 1)
    ON CONFLICT (name, university_id) DO NOTHING;

-- Departments for AITU (university_id = 2)
INSERT INTO departments (name, university_id) VALUES
                                                  ('Information Systems', 2),
                                                  ('Computer Engineering', 2),
                                                  ('Data Science', 2)
    ON CONFLICT (name, university_id) DO NOTHING;

-- Roles
INSERT INTO roles (name, department_id) VALUES
                                            ('Senior Lecturer', 1),
                                            ('Professor', 1),
                                            ('Associate Professor', 1),
                                            ('Head of Department', 1)
    ON CONFLICT (name, department_id) DO NOTHING;

-- System admin user (password: admin123)
INSERT INTO users (email, password, first_name, last_name, role, university_id, department_id)
VALUES (
           'admin@diplomaflow.kz',
           '$2a$14$ajq8Q7fbtFRQvXikDrten.rXMkmy.mQdIT.Z5JtM9.g/q.wj.uAm',
           'System', 'Admin', 'admin', 1, NULL
       )
    ON CONFLICT (email) DO NOTHING;
