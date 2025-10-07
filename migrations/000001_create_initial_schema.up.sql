-- Сначала удаляем старые таблицы, если они существуют, для чистого старта
DROP TABLE IF EXISTS project_reviewers;
DROP TABLE IF EXISTS diploma_projects;
DROP TABLE IF EXISTS staff;
DROP TABLE IF EXISTS students;
DROP TABLE IF EXISTS departments;
DROP TABLE IF EXISTS universities;
DROP TABLE IF EXISTS users;

-- Таблица для всех пользователей системы с разделенными полями имени
CREATE TABLE users (
                       id UUID PRIMARY KEY,
                       email TEXT NOT NULL UNIQUE,
                       password_hash TEXT NOT NULL,
                       last_name TEXT NOT NULL,
                       first_name TEXT NOT NULL,
                       patronymic TEXT, -- Опциональное поле, может быть NULL
                       role VARCHAR(50) NOT NULL
);

CREATE TABLE universities (
                              id UUID PRIMARY KEY,
                              name TEXT NOT NULL UNIQUE
);

CREATE TABLE departments (
                             id UUID PRIMARY KEY,
                             name TEXT NOT NULL,
                             university_id UUID NOT NULL,
                             CONSTRAINT fk_university
                                 FOREIGN KEY(university_id)
                                     REFERENCES universities(id)
                                     ON DELETE CASCADE
);

CREATE TABLE students (
                          user_id UUID PRIMARY KEY,
                          department_id UUID NOT NULL,
                          CONSTRAINT fk_user
                              FOREIGN KEY(user_id)
                                  REFERENCES users(id)
                                  ON DELETE CASCADE,
                          CONSTRAINT fk_department
                              FOREIGN KEY(department_id)
                                  REFERENCES departments(id)
                                  ON DELETE RESTRICT
);

CREATE TABLE staff (
                       user_id UUID PRIMARY KEY,
                       department_id UUID NOT NULL,
                       CONSTRAINT fk_user
                           FOREIGN KEY(user_id)
                               REFERENCES users(id)
                               ON DELETE CASCADE,
                       CONSTRAINT fk_department
                           FOREIGN KEY(department_id)
                               REFERENCES departments(id)
                               ON DELETE RESTRICT
);

-- Остальные таблицы остаются без изменений
CREATE TABLE diploma_projects (
                                  id UUID PRIMARY KEY,
                                  topic TEXT NOT NULL,
                                  status VARCHAR(50) NOT NULL DEFAULT 'draft',
                                  student_id UUID NOT NULL UNIQUE,
                                  supervisor_id UUID NOT NULL,
                                  grade INT,
                                  defense_date TIMESTAMPTZ,
                                  CONSTRAINT fk_student
                                      FOREIGN KEY(student_id)
                                          REFERENCES users(id)
                                          ON DELETE CASCADE,
                                  CONSTRAINT fk_supervisor
                                      FOREIGN KEY(supervisor_id)
                                          REFERENCES users(id)
                                          ON DELETE RESTRICT
);

CREATE TABLE project_reviewers (
                                   project_id UUID NOT NULL,
                                   reviewer_id UUID NOT NULL,
                                   PRIMARY KEY (project_id, reviewer_id),
                                   CONSTRAINT fk_project
                                       FOREIGN KEY(project_id)
                                           REFERENCES diploma_projects(id)
                                           ON DELETE CASCADE,
                                   CONSTRAINT fk_reviewer
                                       FOREIGN KEY(reviewer_id)
                                           REFERENCES users(id)
                                           ON DELETE CASCADE
);

-- Seed Data с обновленной структурой пользователей
INSERT INTO universities (id, name) VALUES
                                        ('11111111-1111-1111-1111-111111111111', 'КазНУ'),
                                        ('22222222-2222-2222-2222-222222222222', 'Назарбаев Университет');

INSERT INTO departments (id, name, university_id) VALUES
                                                      ('aaa11111-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Компьютерные науки', '11111111-1111-1111-1111-111111111111'),
                                                      ('bbb33333-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'Экономика', '22222222-2222-2222-2222-222222222222');

-- Пароль для всех пользователей: "password123"
-- Хеш: $2a$10$g0A.q2H9b1Z8a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u
INSERT INTO users (id, email, password_hash, last_name, first_name, patronymic, role) VALUES
                                                                                          ('stu11111-1111-1111-1111-111111111111', 'ivan@student.kz', '$2a$10$g0A.q2H9b1Z8a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u', 'Иванов', 'Иван', 'Петрович', 'student'),
                                                                                          ('sup11111-1111-1111-1111-111111111111', 'dr.smith@uni.kz', '$2a$10$g0A.q2H9b1Z8a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u', 'Смит', 'Джон', NULL, 'supervisor'),
                                                                                          ('admin1111-1111-1111-1111-111111111111', 'sysadmin@uni.kz', '$2a$10$g0A.q2H9b1Z8a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q.r.s.t.u', 'Админов', 'Систем', NULL, 'sys_admin');

INSERT INTO students (user_id, department_id) VALUES
    ('stu11111-1111-1111-1111-111111111111', 'aaa11111-aaaa-aaaa-aaaa-aaaaaaaaaaaa');

INSERT INTO staff (user_id, department_id) VALUES
    ('sup11111-1111-1111-1111-111111111111', 'aaa11111-aaaa-aaaa-aaaa-aaaaaaaaaaaa');
