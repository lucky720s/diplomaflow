package tests_role

import (
	"context"
	"testing"

	"github.com/lucky720s/diplomaflow/internal/role"
	rolev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/role/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_UpdateRole(t *testing.T) {
	repo := new(MockRepository)
	svc := role.NewService(repo)

	existingRole := &role.Role{ID: 1, Name: "OldName", DepartmentID: 1}

	repo.On("GetByID", mock.Anything, int64(1)).Return(existingRole, nil)

	repo.On("Update", mock.Anything, mock.MatchedBy(func(r *role.Role) bool {
		return r.Name == "NewName" && r.ID == 1
	})).Return(nil)

	res, err := svc.UpdateRole(context.Background(), 1, "NewName", 1, nil)

	require.NoError(t, err)
	require.Equal(t, "NewName", res.Name)
	repo.AssertExpectations(t)
}

func TestHandler_UpdateRole(t *testing.T) {
	mockSvc := new(MockRoleService)
	handler := role.NewHandler(mockSvc)

	updatedRole := &role.Role{ID: 1, Name: "Updated", DepartmentID: 2}

	mockSvc.On("UpdateRole", mock.Anything, int64(1), "Updated", int64(2), mock.Anything).
		Return(updatedRole, nil)

	req := &rolev1.UpdateRoleRequest{
		Role: &rolev1.Role{Id: 1, Name: "Updated", DepartmentId: 2},
	}
	resp, err := handler.UpdateRole(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, "Updated", resp.Role.Name)
}
