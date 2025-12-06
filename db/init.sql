CREATE TABLE IF NOT EXISTS universities (
                                            id BIGSERIAL PRIMARY KEY,
                                            name VARCHAR(255) NOT NULL UNIQUE,
    short_name VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
                             );
CREATE INDEX idx_universities_deleted_at ON universities(deleted_at);

CREATE TABLE IF NOT EXISTS departments (
                                           id BIGSERIAL PRIMARY KEY,
                                           name VARCHAR(255) NOT NULL,
    university_id BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
                             );
CREATE INDEX idx_departments_deleted_at ON departments(deleted_at);
CREATE INDEX idx_departments_university_id ON departments(university_id);

CREATE TABLE IF NOT EXISTS roles (
                                     id BIGSERIAL PRIMARY KEY,
                                     name VARCHAR(255) NOT NULL,
    department_id BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
                             );
CREATE INDEX idx_roles_deleted_at ON roles(deleted_at);

CREATE TABLE IF NOT EXISTS users (
                                     id BIGSERIAL PRIMARY KEY,
                                     email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role VARCHAR(50) NOT NULL,
    university_id BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
                             );
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

INSERT INTO universities (name, short_name) VALUES
    ('Astana IT University', 'AITU');

INSERT INTO departments (name, university_id) VALUES
                                                  ('Computer Science', 1),
                                                  ('Software Engineering', 1);

INSERT INTO roles (name, department_id) VALUES
                                            ('Senior Lecturer', 1),
                                            ('Professor', 1);

-- Пароль для всех: '12345678' (хэш bcrypt cost 14)
INSERT INTO users (email, password, first_name, last_name, role, university_id) VALUES
('student@example.com', '$2a$14$ajq8Q7fbtFRQvXikDrten.rXMkmy.mQdIT.Z5JtM9.g/q.wj.uAm', 'Ivan', 'Ivanov', 'student', 1),
('teacher@example.com', '$2a$14$ajq8Q7fbtFRQvXikDrten.rXMkmy.mQdIT.Z5JtM9.g/q.wj.uAm', 'Petr', 'Petrov', 'teacher', 1),
('admin@example.com', '$2a$14$ajq8Q7fbtFRQvXikDrten.rXMkmy.mQdIT.Z5JtM9.g/q.wj.uAm', 'Admin', 'System', 'admin', 1);
