package tests_role

import (
	"context"

	"github.com/lucky720s/diplomaflow/internal/role"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, r *role.Role) error {
	args := m.Called(ctx, r)
	if r.ID == 0 {
		r.ID = 10
	}
	return args.Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id int64) (*role.Role, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*role.Role), args.Error(1)
}

func (m *MockRepository) Delete(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) Update(ctx context.Context, r *role.Role) error {
	args := m.Called(ctx, r)
	return args.Error(0)
}

func (m *MockRepository) List(ctx context.Context, departmentID int64) ([]*role.Role, error) {
	args := m.Called(ctx, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*role.Role), args.Error(1)
}

type MockRoleService struct {
	mock.Mock
}

func (m *MockRoleService) CreateRole(ctx context.Context, name string, departmentID int64) (*role.Role, error) {
	args := m.Called(ctx, name, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*role.Role), args.Error(1)
}

func (m *MockRoleService) DeleteRole(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRoleService) GetRole(ctx context.Context, id int64) (*role.Role, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*role.Role), args.Error(1)
}

func (m *MockRoleService) UpdateRole(ctx context.Context, id int64, name string, departmentID int64, updateMask *fieldmaskpb.FieldMask) (*role.Role, error) {
	args := m.Called(ctx, id, name, departmentID, updateMask)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*role.Role), args.Error(1)
}

func (m *MockRoleService) ListRoles(ctx context.Context, departmentID int64) ([]*role.Role, error) {
	args := m.Called(ctx, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*role.Role), args.Error(1)
}
