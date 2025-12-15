package tests_auth

import (
	"context"
	"testing"

	"github.com/lucky720s/diplomaflow/internal/auth"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandler_Register_Success(t *testing.T) {
	mockService := new(MockService)
	handler := auth.NewHandler(mockService)

	mockService.
		On("Register", mock.Anything, "test@mail.com", "12345678", "A", "B", "student", int64(1), int64(0)).
		Return(int64(42), nil)

	req := &authv1.RegisterRequest{
		Email:        "test@mail.com",
		Password:     "12345678",
		FirstName:    "A",
		LastName:     "B",
		Role:         "student",
		UniversityId: 1,
	}

	resp, err := handler.Register(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, int64(42), resp.UserId)
}
