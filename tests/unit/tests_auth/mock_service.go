//package tests_auth
//
//import (
//	"context"
//
//	"github.com/lucky720s/diplomaflow/internal/auth"
//	"github.com/stretchr/testify/mock"
//)
//
//type MockService struct {
//	mock.Mock
//}
//
//func (m *MockService) Register(ctx context.Context, email, password, firstName, lastName, role string, universityID int64) (int64, error) {
//	args := m.Called(ctx, email, password, firstName, lastName, role, universityID)
//	return args.Get(0).(int64), args.Error(1)
//}
//
//func (m *MockService) Login(ctx context.Context, email, password string) (string, error) {
//	args := m.Called(ctx, email, password)
//	return args.String(0), args.Error(1)
//}
//
//func (m *MockService) Validate(ctx context.Context, token string) (*auth.JwtClaims, error) {
//	args := m.Called(ctx, token)
//	return args.Get(0).(*auth.JwtClaims), args.Error(1)
//}
//
//func (m *MockService) ListUsers(ctx context.Context, universityID int64, role string, page, pageSize int32) ([]*auth.User, int64, error) {
//	args := m.Called(ctx, universityID, role, page, pageSize)
//	return args.Get(0).([]*auth.User), args.Get(1).(int64), args.Error(2)
//}

package tests_auth

import (
	"context"

	"github.com/lucky720s/diplomaflow/internal/auth"
	"github.com/stretchr/testify/mock"
)

type MockService struct {
	mock.Mock
}

// Register принимает departmentID
func (m *MockService) Register(ctx context.Context, email, password, firstName, lastName, role string, universityID, departmentID int64) (int64, error) {
	args := m.Called(ctx, email, password, firstName, lastName, role, universityID, departmentID)
	// Исправление:
	res, ok := args.Get(0).(int64)
	if !ok {
		panic("args.Get(0) is not int64")
	}
	return res, args.Error(1)
}

// Login принимает userAgent и ip, возвращает 3 значения
func (m *MockService) Login(ctx context.Context, email, password, userAgent, ip string) (string, string, error) {
	args := m.Called(ctx, email, password, userAgent, ip)
	return args.String(0), args.String(1), args.Error(2)
}

// Validate без изменений
func (m *MockService) Validate(ctx context.Context, token string) (*auth.JwtClaims, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	res, ok := args.Get(0).(*auth.JwtClaims)
	if !ok {
		panic("args.Get(0) is not *auth.JwtClaims")
	}
	return res, args.Error(1)
}

// ListUsers принимает excludeUserID
func (m *MockService) ListUsers(ctx context.Context, universityID int64, departmentID int64, role string, page, pageSize int32, excludeUserID int64) ([]*auth.User, int64, error) {
	args := m.Called(ctx, universityID, departmentID, role, page, pageSize, excludeUserID)
	var users []*auth.User
	if args.Get(0) != nil {
		var ok bool
		users, ok = args.Get(0).([]*auth.User)
		if !ok {
			panic("args.Get(0) is not []*auth.User")
		}
	}
	total, ok := args.Get(1).(int64)
	if !ok {
		panic("args.Get(1) is not int64")
	}
	return users, total, args.Error(2)
}

func (m *MockService) AssignRole(ctx context.Context, userID int64, role string) error {
	args := m.Called(ctx, userID, role)
	return args.Error(0)
}

// tests/unit/tests_auth/mock_service.go (или где у тебя MockService)

func (m *MockService) BatchGetUserPreviews(ctx context.Context, ids []int64) ([]*auth.User, error) {
	args := m.Called(ctx, ids)

	var users []*auth.User
	if v := args.Get(0); v != nil {
		var ok bool
		users, ok = v.([]*auth.User)
		if !ok {
			panic("args.Get(0) is not []*auth.User")
		}
	}

	return users, args.Error(1)
}
