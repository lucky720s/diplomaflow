package postgres

import (
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"github.com/lucky720s/diplomaflow/internal/domain"
	apperrors "github.com/lucky720s/diplomaflow/pkg/errors"
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
		return apperrors.WrapErrorf(err, "r.db.ExecContext")
	}
	return nil
}

func (r *DepartmentRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Department, error) {
	query := "SELECT id, name, university_id FROM departments WHERE id = $1"
	var dept domain.Department
	err := r.db.QueryRowContext(ctx, query, id).Scan(&dept.ID, &dept.Name, &dept.UniversityID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, apperrors.WrapErrorf(err, "r.db.QueryRowContext")
	}
	return &dept, nil
}
