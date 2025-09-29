-- Удаляем в обратном порядке, чтобы не нарушать возможные ограничения целостности
DELETE FROM students WHERE id IN ('stud-001', 'stud-002', 'stud-003', 'stud-004', 'stud-005');
DELETE FROM departments WHERE id IN ('dep-it', 'dep-econ', 'dep-arch');
