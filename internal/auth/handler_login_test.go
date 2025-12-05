package auth

import (
	"context"
	"testing"

	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandler_Login_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	mockService.
		On("Login", mock.Anything, "test@mail.com", "12345").
		Return("token123", nil)

	req := &authv1.LoginRequest{
		Email:    "test@mail.com",
		Password: "12345",
	}

	resp, err := handler.Login(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, "token123", resp.Token)
}
