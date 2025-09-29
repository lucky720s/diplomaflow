package domain

import (
	"context"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
}

type StudentRepository interface {
	CreateProfile(ctx context.Context, userID, departmentID uuid.UUID) error
	GetProfile(ctx context.Context, userID uuid.UUID) (*StudentProfile, error)
}

type DepartmentRepository interface {
	Create(ctx context.Context, department *Department) error
	GetByID(ctx context.Context, id uuid.UUID) (*Department, error)
}
