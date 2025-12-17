package tests_role

import (
	"context"
	"testing"

	"github.com/lucky720s/diplomaflow/internal/role"
	rolev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/role/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_DeleteRole(t *testing.T) {
	repo := new(MockRepository)
	svc := role.NewService(repo)

	repo.On("Delete", mock.Anything, int64(10)).Return(nil)

	err := svc.DeleteRole(context.Background(), 10)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestHandler_DeleteRole(t *testing.T) {
	mockSvc := new(MockRoleService)
	handler := role.NewHandler(mockSvc)

	mockSvc.On("DeleteRole", mock.Anything, int64(10)).Return(nil)

	req := &rolev1.DeleteRoleRequest{RoleId: 10}
	resp, err := handler.DeleteRole(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
}
