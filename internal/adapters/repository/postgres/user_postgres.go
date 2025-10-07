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
	query := "INSERT INTO users (id, email, password_hash, last_name, first_name, patronymic, role) VALUES ($1, $2, $3, $4, $5, $6, $7)"
	_, err := r.db.ExecContext(ctx, query, user.ID, user.Email, user.PasswordHash, user.LastName, user.FirstName, user.Patronymic, user.Role)
	if err != nil {
		return apperrors.WrapErrorf(err, "r.db.ExecContext")
	}
	return nil
}

func (r *UserRepo) scanUser(row interface{ Scan(...interface{}) error }) (*domain.User, error) {
	var user domain.User
	var patronymic sql.NullString

	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.LastName, &user.FirstName, &patronymic, &user.Role)
	if err != nil {
		return nil, err
	}

	if patronymic.Valid {
		user.Patronymic = &patronymic.String
	}

	return &user, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := "SELECT id, email, password_hash, last_name, first_name, patronymic, role FROM users WHERE email = $1"
	row := r.db.QueryRowContext(ctx, query, email)

	user, err := r.scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, apperrors.WrapErrorf(err, "r.scanUser")
	}
	return user, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := "SELECT id, email, password_hash, last_name, first_name, patronymic, role FROM users WHERE id = $1"
	row := r.db.QueryRowContext(ctx, query, id)

	user, err := r.scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, apperrors.WrapErrorf(err, "r.scanUser")
	}
	return user, nil
}
