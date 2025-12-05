package auth

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) Register(ctx context.Context, email, password, firstName, lastName, role string, universityID int64) (int64, error) {
	args := m.Called(ctx, email, password, firstName, lastName, role, universityID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockService) Login(ctx context.Context, email, password string) (string, error) {
	args := m.Called(ctx, email, password)
	return args.String(0), args.Error(1)
}

func (m *MockService) Validate(ctx context.Context, token string) (*JwtClaims, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(*JwtClaims), args.Error(1)
}

func (m *MockService) ListUsers(ctx context.Context, universityID int64, role string, page, pageSize int32) ([]*User, int64, error) {
	args := m.Called(ctx, universityID, role, page, pageSize)
	return args.Get(0).([]*User), args.Get(1).(int64), args.Error(2)
}
