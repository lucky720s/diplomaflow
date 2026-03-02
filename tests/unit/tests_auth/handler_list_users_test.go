package tests_auth

import (
	"context"
	"testing"

	"github.com/lucky720s/diplomaflow/internal/auth"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestHandler_ListUsers_Success(t *testing.T) {
	mockService := new(MockService)
	handler := auth.NewHandler(mockService)

	users := []*auth.User{
		{ID: 1, Email: "a@mail.com", FirstName: "A", LastName: "A", Role: "student", UniversityID: 1, DepartmentID: 10},
		{ID: 2, Email: "b@mail.com", FirstName: "B", LastName: "B", Role: "student", UniversityID: 1, DepartmentID: 10},
	}

	// ВАЖНО: ожидание должно совпасть с реальным вызовом из panic:
	// univ=1, dept=0, pageSize=20
	mockService.
		On("ListUsers",
			mock.Anything, // ctx (НЕ "mock.Anything")
			int64(1),      // universityID
			int64(0),      // departmentID
			"student",     // role filter
			int32(1),      // page
			int32(20),     // pageSize
			int64(0),      // excludeUserID
		).
		Return(users, int64(2), nil).
		Once()

	req := &authv1.ListUsersRequest{
		UniversityId:  1,
		DepartmentId:  0, // admin может 0 => "все кафедры"
		Role:          "student",
		Page:          1,
		PageSize:      20,
		ExcludeUserId: 0,
	}

	// Handler.ListUsers требует metadata (x-internal-service и роль) [[12]]
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-internal-service", "api_gateway",
		"x-user-role", "admin",
		"x-user-id", "999",
		"x-university-id", "1",
		"x-department-id", "10",
	))

	resp, err := handler.ListUsers(ctx, req)

	require.NoError(t, err)
	require.Equal(t, int64(2), resp.TotalCount)
	require.Len(t, resp.Users, 2)

	mockService.AssertExpectations(t)
}
