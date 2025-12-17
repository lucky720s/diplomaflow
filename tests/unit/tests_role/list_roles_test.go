package tests_role

import (
	"context"
	"testing"

	"github.com/lucky720s/diplomaflow/internal/role"
	rolev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/role/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_ListRoles(t *testing.T) {
	repo := new(MockRepository)
	svc := role.NewService(repo)

	roles := []*role.Role{{ID: 1}, {ID: 2}}
	repo.On("List", mock.Anything, int64(5)).Return(roles, nil)

	res, err := svc.ListRoles(context.Background(), 5)

	require.NoError(t, err)
	require.Len(t, res, 2)
}

func TestHandler_ListRoles(t *testing.T) {
	mockSvc := new(MockRoleService)
	handler := role.NewHandler(mockSvc)

	roles := []*role.Role{{ID: 1}, {ID: 2}}
	mockSvc.On("ListRoles", mock.Anything, int64(5)).Return(roles, nil)

	req := &rolev1.ListRolesRequest{DepartmentId: 5}
	resp, err := handler.ListRoles(context.Background(), req)

	require.NoError(t, err)
	require.Len(t, resp.Roles, 2)
}
