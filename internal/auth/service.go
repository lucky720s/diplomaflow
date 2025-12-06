package auth

import (
	"context"
	"errors"

	rolev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/role/v1"
	universityv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/university/v1"
	"gorm.io/gorm"
)

type Service struct {
	repo       Repository
	jwtWrapper JwtWrapper
	univClient universityv1.UniversityServiceClient
	roleClient rolev1.RoleServiceClient
}

func NewService(
	repo Repository,
	jwtWrapper JwtWrapper,
	univClient universityv1.UniversityServiceClient,
	roleClient rolev1.RoleServiceClient,
) *Service {
	return &Service{
		repo:       repo,
		jwtWrapper: jwtWrapper,
		univClient: univClient,
		roleClient: roleClient,
	}
}

func (s *Service) Register(ctx context.Context, email, password, firstName, lastName, role string, universityID, departmentID int64) (int64, error) {
	existingUser, err := s.repo.GetByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return 0, errors.New("user already exists")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	hashedPassword, err := HashPassword(password)
	if err != nil {
		return 0, err
	}

	user := &User{
		Email:        email,
		Password:     hashedPassword,
		FirstName:    firstName,
		LastName:     lastName,
		Role:         role,
		UniversityID: universityID,
		DepartmentID: departmentID,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return 0, err
	}

	return user.ID, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("user not found")
		}
		return "", err
	}

	if !CheckPasswordHash(password, user.Password) {
		return "", errors.New("invalid credentials")
	}

	token, err := s.jwtWrapper.GenerateToken(*user)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *Service) Validate(ctx context.Context, token string) (*JwtClaims, error) {
	return s.jwtWrapper.ValidateToken(token)
}

func (s *Service) ListUsers(ctx context.Context, universityID int64, role string, page, pageSize int32) ([]*User, int64, error) {
	if pageSize <= 0 {
		pageSize = 10
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize

	filter := UserFilter{
		UniversityID: universityID,
		Role:         role,
		Limit:        int(pageSize),
		Offset:       int(offset),
	}

	return s.repo.ListUsers(ctx, filter)
}
