// file: internal/adapters/repository/postgres/student_postgres.go
package postgres

import (
	"context"
	"database/sql"
	"github.com/lucky720s/diplomaflow/internal/domain"
	"github.com/lucky720s/diplomaflow/pkg/errors"
)

// StudentRepo - реализация репозитория студентов для PostgreSQL
type StudentRepo struct {
	db *sql.DB // Или *gorm.DB, если будете использовать GORM
}

// NewStudentRepo создает новый экземпляр репозитория
func NewStudentRepo(db *sql.DB) *StudentRepo {
	return &StudentRepo{db: db}
}

// Create реализует метод интерфейса domain.StudentRepository
func (r *StudentRepo) Create(ctx context.Context, student *domain.Student) error {
	query := "INSERT INTO students (id, full_name, department_id) VALUES ($1, $2, $3)"

	_, err := r.db.ExecContext(ctx, query, student.ID, student.FullName, student.DepartmentID)
	if err != nil {
		return errors.WrapErrorf(err, "r.db.ExecContext")
	}

	return nil
}

// GetByID ...

func (r *StudentRepo) GetByID(ctx context.Context, id string) (*domain.Student, error) {
	query := "SELECT id, full_name, department_id FROM students WHERE id = $1"

	var student domain.Student
	err := r.db.QueryRowContext(ctx, query, id).Scan(&student.ID, &student.FullName, &student.DepartmentID)
	if err != nil {
		// Важно обрабатывать случай, когда студент не найден
		if err == sql.ErrNoRows {
			// Используйте свой пакет ошибок для консистентности
			return nil, errors.WrapErrorf(err, "student with id %s not found", id)
		}
		return nil, errors.WrapErrorf(err, "r.db.QueryRowContext")
	}

	return &student, nil
}

// List возвращает список всех студентов
func (r *StudentRepo) List(ctx context.Context) ([]*domain.Student, error) {
	query := "SELECT id, full_name, department_id FROM students"

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, errors.WrapErrorf(err, "r.db.QueryContext")
	}
	defer rows.Close()

	var students []*domain.Student
	for rows.Next() {
		var student domain.Student
		if err := rows.Scan(&student.ID, &student.FullName, &student.DepartmentID); err != nil {
			return nil, errors.WrapErrorf(err, "rows.Scan")
		}
		students = append(students, &student)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.WrapErrorf(err, "rows.Err")
	}

	return students, nil
}

// Delete удаляет студента по ID
func (r *StudentRepo) Delete(ctx context.Context, id string) error {
	query := "DELETE FROM students WHERE id = $1"

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return errors.WrapErrorf(err, "r.db.ExecContext")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.WrapErrorf(err, "result.RowsAffected")
	}

	if rowsAffected == 0 {
		return errors.WrapErrorf(sql.ErrNoRows, "student with id %s not found for deletion", id)
	}

	return nil
}
