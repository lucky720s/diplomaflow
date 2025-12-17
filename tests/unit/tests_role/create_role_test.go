package tests_role

import (
	"context"
	"testing"

	"github.com/lucky720s/diplomaflow/internal/role"
	rolev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/role/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_CreateRole(t *testing.T) {
	repo := new(MockRepository)
	svc := role.NewService(repo)

	repo.On("Create", mock.Anything, mock.MatchedBy(func(r *role.Role) bool {
		return r.Name == "Admin" && r.DepartmentID == 1
	})).Return(nil)

	res, err := svc.CreateRole(context.Background(), "Admin", 1)

	require.NoError(t, err)
	require.Equal(t, int64(10), res.ID)
	repo.AssertExpectations(t)
}

func TestHandler_CreateRole(t *testing.T) {
	mockSvc := new(MockRoleService)
	handler := role.NewHandler(mockSvc)

	expectedRole := &role.Role{ID: 100, Name: "Admin", DepartmentID: 5}

	mockSvc.On("CreateRole", mock.Anything, "Admin", int64(5)).
		Return(expectedRole, nil)

	req := &rolev1.CreateRoleRequest{Name: "Admin", DepartmentId: 5}
	resp, err := handler.CreateRole(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, int64(100), resp.Role.Id)
	require.Equal(t, "Admin", resp.Role.Name)
}
