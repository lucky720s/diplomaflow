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
