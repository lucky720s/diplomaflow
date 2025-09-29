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
	// Реализация поиска студента
	return nil, nil // TODO
}
