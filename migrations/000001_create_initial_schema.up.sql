CREATE TABLE users (
                       id UUID PRIMARY KEY,
                       email TEXT NOT NULL UNIQUE,
                       password_hash TEXT NOT NULL,
                       full_name TEXT NOT NULL,
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
-- Вставляем университеты
INSERT INTO universities (id, name) VALUES
                                        (gen_random_uuid(), 'КазНУ им. аль-Фараби'),
                                        (gen_random_uuid(), 'Satbayev University');

-- Вставляем кафедры
INSERT INTO departments (id, name, university_id) VALUES
                                                      (gen_random_uuid(), 'Информатика', (SELECT id FROM universities WHERE name = 'КазНУ им. аль-Фараби')),
                                                      (gen_random_uuid(), 'Математика', (SELECT id FROM universities WHERE name = 'Satbayev University'));

-- Вставляем пользователей (пароль пока в виде "hash123", потом заменим на bcrypt)
INSERT INTO users (id, email, password_hash, full_name, role) VALUES
                                                                  (gen_random_uuid(), 'student1@example.com', 'hash123', 'Иван Иванов', 'student'),
                                                                  (gen_random_uuid(), 'student2@example.com', 'hash123', 'Алия Садыкова', 'student'),
                                                                  (gen_random_uuid(), 'staff1@example.com', 'hash123', 'Проф. Ерлан Касымов', 'staff'),
                                                                  (gen_random_uuid(), 'staff2@example.com', 'hash123', 'Доцент Гульмира Ахметова', 'staff'),
                                                                  (gen_random_uuid(), 'reviewer1@example.com', 'hash123', 'Рецензент Асель Куаныш', 'staff');

-- Привязываем студентов к кафедрам
INSERT INTO students (user_id, department_id) VALUES
                                                  ((SELECT id FROM users WHERE email = 'student1@example.com'),
                                                   (SELECT id FROM departments WHERE name = 'Информатика')),
                                                  ((SELECT id FROM users WHERE email = 'student2@example.com'),
                                                   (SELECT id FROM departments WHERE name = 'Математика'));

-- Привязываем сотрудников к кафедрам
INSERT INTO staff (user_id, department_id) VALUES
                                               ((SELECT id FROM users WHERE email = 'staff1@example.com'),
                                                (SELECT id FROM departments WHERE name = 'Информатика')),
                                               ((SELECT id FROM users WHERE email = 'staff2@example.com'),
                                                (SELECT id FROM departments WHERE name = 'Математика')),
                                               ((SELECT id FROM users WHERE email = 'reviewer1@example.com'),
                                                (SELECT id FROM departments WHERE name = 'Информатика'));

-- Дипломные проекты
INSERT INTO diploma_projects (id, topic, status, student_id, supervisor_id, grade, defense_date) VALUES
                                                                                                     (gen_random_uuid(), 'Разработка системы управления дипломными проектами', 'in_progress',
                                                                                                      (SELECT id FROM users WHERE email = 'student1@example.com'),
                                                                                                      (SELECT id FROM users WHERE email = 'staff1@example.com'),
                                                                                                      NULL, NULL),
                                                                                                     (gen_random_uuid(), 'Моделирование математических процессов', 'draft',
                                                                                                      (SELECT id FROM users WHERE email = 'student2@example.com'),
                                                                                                      (SELECT id FROM users WHERE email = 'staff2@example.com'),
                                                                                                      NULL, NULL);

-- Рецензенты для проектов
INSERT INTO project_reviewers (project_id, reviewer_id) VALUES
    ((SELECT id FROM diploma_projects WHERE topic = 'Разработка системы управления дипломными проектами'),
     (SELECT id FROM users WHERE email = 'reviewer1@example.com'));
