package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/lucky720s/diplomaflow/internal/domain"
	apperrors "github.com/lucky720s/diplomaflow/pkg/errors"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *domain.User) error {
	query := "INSERT INTO users (id, email, password_hash, full_name, role) VALUES ($1, $2, $3, $4, $5)"
	_, err := r.db.ExecContext(ctx, query, user.ID, user.Email, user.PasswordHash, user.FullName, user.Role)
	if err != nil {
		return apperrors.WrapErrorf(err, "r.db.ExecContext")
	}
	return nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := "SELECT id, email, password_hash, full_name, role FROM users WHERE email = $1"
	var user domain.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.Role)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, apperrors.WrapErrorf(err, "r.db.QueryRowContext")
	}
	return &user, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := "SELECT id, email, password_hash, full_name, role FROM users WHERE id = $1"
	var user domain.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.Role)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, apperrors.WrapErrorf(err, "r.db.QueryRowContext")
	}
	return &user, nil
}
