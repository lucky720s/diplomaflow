package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/lucky720s/diplomaflow/internal/domain"
	apperrors "github.com/lucky720s/diplomaflow/pkg/errors"
)

type StudentRepo struct {
	db *sql.DB
}

func NewStudentRepo(db *sql.DB) *StudentRepo {
	return &StudentRepo{db: db}
}

func (r *StudentRepo) CreateProfile(ctx context.Context, userID, departmentID uuid.UUID) error {
	query := "INSERT INTO students (user_id, department_id) VALUES ($1, $2)"
	_, err := r.db.ExecContext(ctx, query, userID, departmentID)
	if err != nil {
		return apperrors.WrapErrorf(err, "r.db.ExecContext")
	}
	return nil
}

func (r *StudentRepo) GetProfile(ctx context.Context, userID uuid.UUID) (*domain.StudentProfile, error) {
	query := `
		SELECT u.id, u.email, u.full_name, u.role, d.id, d.name, d.university_id
		FROM users u
		JOIN students s ON u.id = s.user_id
		JOIN departments d ON s.department_id = d.id
		WHERE u.id = $1
	`
	var profile domain.StudentProfile
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&profile.User.ID,
		&profile.User.Email,
		&profile.User.FullName,
		&profile.User.Role,
		&profile.Department.ID,
		&profile.Department.Name,
		&profile.Department.UniversityID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, apperrors.WrapErrorf(err, "r.db.QueryRowContext")
	}
	return &profile, nil
}
