package tests_auth

import (
	"context"
	"testing"

	"github.com/lucky720s/diplomaflow/internal/auth"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

/*wefnwnfnfw*/

func TestHandler_ListUsers_Success(t *testing.T) {
	mockService := new(MockService)
	handler := auth.NewHandler(mockService)

	users := []*auth.User{
		{ID: 1, Email: "a@mail.com", FirstName: "A", LastName: "A", Role: "student", UniversityID: 1, DepartmentID: 1},
		{ID: 2, Email: "bbbbb44444@mail.com", FirstName: "B", LastName: "B", Role: "teacher", UniversityID: 1, DepartmentID: 1},
	}

	// ИСПРАВЛЕНИЕ: Добавлен int64(0) в конце (это excludeUserID)
	mockService.
		On("ListUsers",
			mock.Anything,
			int64(1),  // universityID
			int64(1),  // departmentID
			"student", // role
			int32(1),  // page
			int32(10), // pageSize
			int64(0),  // excludeUserID
		).
		Return(users, int64(2), nil)

	req := &authv1.ListUsersRequest{
		UniversityId: 1,
		DepartmentId: 1,
		Role:         "student",
		Page:         1,
		PageSize:     10,
	}

	resp, err := handler.ListUsers(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, int64(2), resp.TotalCount)
	require.Len(t, resp.Users, 2)
}
