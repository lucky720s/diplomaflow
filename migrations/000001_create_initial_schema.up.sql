-- Таблица для всех пользователей системы
CREATE TABLE users (
                       id UUID PRIMARY KEY,
                       email TEXT NOT NULL UNIQUE,
                       password_hash TEXT NOT NULL,
                       full_name TEXT NOT NULL,
                       role VARCHAR(50) NOT NULL -- 'student', 'supervisor', 'dept_admin', 'sys_admin'
);

-- Таблица университетов
CREATE TABLE universities (
                              id UUID PRIMARY KEY,
                              name TEXT NOT NULL UNIQUE
);

-- Таблица кафедр, привязанных к университетам
CREATE TABLE departments (
                             id UUID PRIMARY KEY,
                             name TEXT NOT NULL,
                             university_id UUID NOT NULL,
                             CONSTRAINT fk_university
                                 FOREIGN KEY(university_id)
                                     REFERENCES universities(id)
                                     ON DELETE CASCADE -- Если удаляем университет, удаляем и его кафедры
);

-- Таблица, расширяющая пользователей до роли "Студент"
CREATE TABLE students (
                          user_id UUID PRIMARY KEY, -- Это и первичный, и внешний ключ
                          department_id UUID NOT NULL,
                          CONSTRAINT fk_user
                              FOREIGN KEY(user_id)
                                  REFERENCES users(id)
                                  ON DELETE CASCADE, -- Если удаляем пользователя, удаляем и запись о студенте
                          CONSTRAINT fk_department
                              FOREIGN KEY(department_id)
                                  REFERENCES departments(id)
                                  ON DELETE RESTRICT -- Нельзя удалить кафедру, если на ней есть студенты
);

-- Таблица, расширяющая пользователей до роли "Сотрудник" (руководитель, админ)
CREATE TABLE staff (
                       user_id UUID PRIMARY KEY, -- Это и первичный, и внешний ключ
                       department_id UUID NOT NULL,
                       CONSTRAINT fk_user
                           FOREIGN KEY(user_id)
                               REFERENCES users(id)
                               ON DELETE CASCADE, -- Если удаляем пользователя, удаляем и запись о сотруднике
                       CONSTRAINT fk_department
                           FOREIGN KEY(department_id)
                               REFERENCES departments(id)
                               ON DELETE RESTRICT -- Нельзя удалить кафедру, если на ней есть сотрудники
);

-- Таблица дипломных проектов
CREATE TABLE diploma_projects (
                                  id UUID PRIMARY KEY,
                                  topic TEXT NOT NULL,
                                  status VARCHAR(50) NOT NULL DEFAULT 'draft',
                                  student_id UUID NOT NULL UNIQUE, -- У одного студента один проект
                                  supervisor_id UUID NOT NULL,
                                  grade INT,
                                  defense_date TIMESTAMPTZ,
                                  CONSTRAINT fk_student
                                      FOREIGN KEY(student_id)
                                          REFERENCES users(id)
                                          ON DELETE CASCADE, -- Если удаляем студента, удаляем и его проект
                                  CONSTRAINT fk_supervisor
                                      FOREIGN KEY(supervisor_id)
                                          REFERENCES users(id)
                                          ON DELETE RESTRICT -- Нельзя удалить руководителя, если у него есть проекты
);

-- Таблица-связка для рецензентов (Многие-ко-Многим)
CREATE TABLE project_reviewers (
                                   project_id UUID NOT NULL,
                                   reviewer_id UUID NOT NULL,
                                   PRIMARY KEY (project_id, reviewer_id), -- Составной первичный ключ
                                   CONSTRAINT fk_project
                                       FOREIGN KEY(project_id)
                                           REFERENCES diploma_projects(id)
                                           ON DELETE CASCADE, -- Если удаляем проект, удаляем и записи о рецензентах
                                   CONSTRAINT fk_reviewer
                                       FOREIGN KEY(reviewer_id)
                                           REFERENCES users(id)
                                           ON DELETE CASCADE -- Если удаляем пользователя-рецензента, удаляем его из проектов
);
