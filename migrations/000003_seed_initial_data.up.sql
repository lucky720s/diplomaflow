-- Сначала добавляем кафедры
INSERT INTO departments (id, name, university_id) VALUES
                                                      ('dep-it', 'Кафедра Информационных Технологий', 'bstu-main'),
                                                      ('dep-econ', 'Кафедра Экономики и Управления', 'bstu-main'),
                                                      ('dep-arch', 'Кафедра Архитектуры и Градостроительства', 'bstu-main');

-- Теперь добавляем студентов, ссылаясь на ID кафедр
INSERT INTO students (id, full_name, department_id) VALUES
                                                        ('stud-001', 'Иванов Иван Иванович', 'dep-it'),
                                                        ('stud-002', 'Петрова Мария Сергеевна', 'dep-it'),
                                                        ('stud-003', 'Сидоров Алексей Петрович', 'dep-econ'),
                                                        ('stud-004', 'Кузнецова Ольга Владимировна', 'dep-arch'),
                                                        ('stud-005', 'Васильев Дмитрий Андреевич', 'dep-econ');
