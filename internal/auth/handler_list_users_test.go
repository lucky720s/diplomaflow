package auth

import (
	"context"
	"testing"

	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandler_ListUsers_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	users := []*User{
		{ID: 1, Email: "a@mail.com", FirstName: "A", LastName: "A", Role: "student", UniversityID: 1},
		{ID: 2, Email: "b@mail.com", FirstName: "B", LastName: "B", Role: "teacher", UniversityID: 1},
	}

	mockService.
		On("ListUsers", mock.Anything, int64(1), "student", int32(1), int32(10)).
		Return(users, int64(2), nil)

	req := &authv1.ListUsersRequest{
		UniversityId: 1,
		Role:         "student",
		Page:         1,
		PageSize:     10,
	}

	resp, err := handler.ListUsers(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, int64(2), resp.TotalCount)
	require.Len(t, resp.Users, 2)
}
