package usecase

import (
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"github.com/lucky720s/diplomaflow/internal/auth"
	"github.com/lucky720s/diplomaflow/internal/domain"
	apperrors "github.com/lucky720s/diplomaflow/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserExists    = errors.New("user with this email already exists")
	ErrDeptNotFound  = errors.New("department not found")
	ErrInvalidCreds  = errors.New("invalid email or password")
	ErrForbiddenRole = errors.New("this role cannot be registered via this endpoint")
)

type AuthUsecase struct {
	userRepo    domain.UserRepository
	studentRepo domain.StudentRepository
	deptRepo    domain.DepartmentRepository
}

func NewAuthUsecase(userRepo domain.UserRepository, studentRepo domain.StudentRepository, deptRepo domain.DepartmentRepository) *AuthUsecase {
	return &AuthUsecase{
		userRepo:    userRepo,
		studentRepo: studentRepo,
		deptRepo:    deptRepo,
	}
}

type RegisterUserInput struct {
	Email        string
	Password     string
	FullName     string
	Role         domain.Role
	DepartmentID uuid.UUID
}

func (uc *AuthUsecase) Register(ctx context.Context, input RegisterUserInput) (*domain.User, error) {
	if _, err := uc.userRepo.GetByEmail(ctx, input.Email); !errors.Is(err, sql.ErrNoRows) {
		if err == nil {
			return nil, ErrUserExists
		}
		return nil, apperrors.WrapErrorf(err, "uc.userRepo.GetByEmail")
	}

	if _, err := uc.deptRepo.GetByID(ctx, input.DepartmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeptNotFound
		}
		return nil, apperrors.WrapErrorf(err, "uc.deptRepo.GetByID")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperrors.WrapErrorf(err, "bcrypt.GenerateFromPassword")
	}

	user := &domain.User{
		ID:           uuid.New(),
		Email:        input.Email,
		PasswordHash: string(hashedPassword),
		FullName:     input.FullName,
		Role:         input.Role,
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, apperrors.WrapErrorf(err, "uc.userRepo.Create")
	}

	switch user.Role {
	case domain.RoleStudent:
		if err := uc.studentRepo.CreateProfile(ctx, user.ID, input.DepartmentID); err != nil {
			return nil, apperrors.WrapErrorf(err, "uc.studentRepo.CreateProfile")
		}
	case domain.RoleSupervisor, domain.RoleDeptAdmin, domain.RoleSysAdmin:
		// TODO: Implement staff profile creation in a separate usecase
	default:
		return nil, ErrForbiddenRole
	}

	return user, nil
}

func (uc *AuthUsecase) Login(ctx context.Context, email, password string) (string, error) {
	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrInvalidCreds
		}
		return "", apperrors.WrapErrorf(err, "uc.userRepo.GetByEmail")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCreds
	}

	token, err := auth.GenerateToken(user)
	if err != nil {
		return "", apperrors.WrapErrorf(err, "auth.GenerateToken")
	}

	return token, nil
}
