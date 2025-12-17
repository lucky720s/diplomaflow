package tests_role

import (
	"context"
	"testing"

	"github.com/lucky720s/diplomaflow/internal/role"
	rolev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/role/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_GetRole(t *testing.T) {
	repo := new(MockRepository)
	svc := role.NewService(repo)

	expectedRole := &role.Role{ID: 1, Name: "User"}
	repo.On("GetByID", mock.Anything, int64(1)).Return(expectedRole, nil)

	res, err := svc.GetRole(context.Background(), 1)

	require.NoError(t, err)
	require.Equal(t, expectedRole, res)
}

func TestHandler_GetRole(t *testing.T) {
	mockSvc := new(MockRoleService)
	handler := role.NewHandler(mockSvc)

	expectedRole := &role.Role{ID: 1, Name: "User"}
	mockSvc.On("GetRole", mock.Anything, int64(1)).Return(expectedRole, nil)

	req := &rolev1.GetRoleRequest{RoleId: 1}
	resp, err := handler.GetRole(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, "User", resp.Role.Name)
}
