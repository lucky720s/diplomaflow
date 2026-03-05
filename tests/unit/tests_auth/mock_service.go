package tests_auth

import (
	"context"

	"github.com/lucky720s/diplomaflow/internal/auth"
	"github.com/stretchr/testify/mock"
)

type MockService struct {
	mock.Mock
}

// ==================== existing methods ====================

func (m *MockService) Register(ctx context.Context, email, password, firstName, lastName, role string, universityID, departmentID int64) (int64, error) {
	args := m.Called(ctx, email, password, firstName, lastName, role, universityID, departmentID)
	res, ok := args.Get(0).(int64)
	if !ok {
		panic("args.Get(0) is not int64")
	}
	return res, args.Error(1)
}

func (m *MockService) Login(ctx context.Context, email, password, userAgent, ip string) (string, string, error) {
	args := m.Called(ctx, email, password, userAgent, ip)
	return args.String(0), args.String(1), args.Error(2)
}

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

func (m *MockService) ListUsers(ctx context.Context, universityID int64, departmentID int64, role string, page, pageSize int32, excludeUserID int64) ([]*auth.User, int64, error) {
	args := m.Called(ctx, universityID, departmentID, role, page, pageSize, excludeUserID)

	var users []*auth.User
	if v := args.Get(0); v != nil {
		var ok bool
		users, ok = v.([]*auth.User)
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

// ==================== NEW: dynamic department roles ====================

func (m *MockService) CreateDepartmentRole(ctx context.Context, departmentID int64, slug string) (*auth.DepartmentRole, error) {
	args := m.Called(ctx, departmentID, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	res, ok := args.Get(0).(*auth.DepartmentRole)
	if !ok {
		panic("args.Get(0) is not *auth.DepartmentRole")
	}
	return res, args.Error(1)
}

func (m *MockService) ListDepartmentRoles(ctx context.Context, departmentID int64) ([]*auth.DepartmentRole, error) {
	args := m.Called(ctx, departmentID)
	var roles []*auth.DepartmentRole
	if v := args.Get(0); v != nil {
		var ok bool
		roles, ok = v.([]*auth.DepartmentRole)
		if !ok {
			panic("args.Get(0) is not []*auth.DepartmentRole")
		}
	}
	return roles, args.Error(1)
}

func (m *MockService) GetDepartmentRole(ctx context.Context, roleID int64) (*auth.DepartmentRole, error) {
	args := m.Called(ctx, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	res, ok := args.Get(0).(*auth.DepartmentRole)
	if !ok {
		panic("args.Get(0) is not *auth.DepartmentRole")
	}
	return res, args.Error(1)
}

func (m *MockService) DeleteDepartmentRole(ctx context.Context, roleID int64) error {
	args := m.Called(ctx, roleID)
	return args.Error(0)
}

func (m *MockService) AssignDepartmentRole(ctx context.Context, userID, departmentID, roleID int64, assignedBy int64, comment string) error {
	args := m.Called(ctx, userID, departmentID, roleID, assignedBy, comment)
	return args.Error(0)
}

func (m *MockService) RevokeDepartmentRole(ctx context.Context, userID, departmentID, roleID int64, revokedBy int64, comment string) error {
	args := m.Called(ctx, userID, departmentID, roleID, revokedBy, comment)
	return args.Error(0)
}

func (m *MockService) ListUserDepartmentRoleSlugs(ctx context.Context, userID, departmentID int64) ([]string, error) {
	args := m.Called(ctx, userID, departmentID)

	var slugs []string
	if v := args.Get(0); v != nil {
		var ok bool
		slugs, ok = v.([]string)
		if !ok {
			panic("args.Get(0) is not []string")
		}
	}

	return slugs, args.Error(1)
}
func (m *MockService) DeleteUser(ctx context.Context, userID, requesterID int64) error {
	return nil
}
func (m *MockService) GetUser(ctx context.Context, userID int64) (*auth.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	res, ok := args.Get(0).(*auth.User)
	if !ok {
		panic("args.Get(0) is not *auth.User")
	}
	return res, args.Error(1)
}

func (m *MockService) UpdateUser(ctx context.Context, userID int64, email, firstName, lastName, role string, universityID, departmentID int64, password string) (*auth.User, error) {
	args := m.Called(ctx, userID, email, firstName, lastName, role, universityID, departmentID, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	res, ok := args.Get(0).(*auth.User)
	if !ok {
		panic("args.Get(0) is not *auth.User")
	}
	return res, args.Error(1)
}
