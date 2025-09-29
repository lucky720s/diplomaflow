package postgres

import (
	"context"
	"database/sql"
	"github.com/lucky720s/diplomaflow/internal/domain"
	"github.com/lucky720s/diplomaflow/pkg/errors"
)

type DepartmentRepo struct {
	db *sql.DB
}

func NewDepartmentRepo(db *sql.DB) *DepartmentRepo {
	return &DepartmentRepo{db: db}
}

func (r *DepartmentRepo) Create(ctx context.Context, department *domain.Department) error {
	query := "INSERT INTO departments (id, name, university_id) VALUES ($1, $2, $3)"

	_, err := r.db.ExecContext(ctx, query, department.ID, department.Name, department.UniversityID)
	if err != nil {
		return errors.WrapErrorf(err, "r.db.ExecContext")
	}

	return nil
}

func (r *DepartmentRepo) GetByID(ctx context.Context, id string) (*domain.Department, error) {
	query := "SELECT id, name, university_id FROM departments WHERE id = $1"

	var dept domain.Department
	err := r.db.QueryRowContext(ctx, query, id).Scan(&dept.ID, &dept.Name, &dept.UniversityID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.WrapErrorf(err, "department with id %s not found", id)
		}
		return nil, errors.WrapErrorf(err, "r.db.QueryRowContext")
	}

	return &dept, nil
}
