package tests_auth

import (
	"context"
	"testing"

	"github.com/lucky720s/diplomaflow/internal/auth"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandler_Login_Success(t *testing.T) {
	mockService := new(MockService)
	handler := auth.NewHandler(mockService)

	// ИСПРАВЛЕНИЕ:
	// 1. Добавлены mock.Anything, mock.Anything (UserAgent и IP)
	// 2. Возвращаем ("access_token", "refresh_token", nil)
	mockService.
		On("Login", mock.Anything, "test@mail.com", "12345", mock.Anything, mock.Anything).
		Return("access_token_123", "refresh_token_123", nil)

	req := &authv1.LoginRequest{
		Email:    "test@mail.com",
		Password: "12345",
	}

	resp, err := handler.Login(context.Background(), req)

	require.NoError(t, err)
	// Проверяем AccessToken (поле Token удалено из proto)
	require.Equal(t, "access_token_123", resp.AccessToken)
	// Можно проверить и RefreshToken
	require.Equal(t, "refresh_token_123", resp.RefreshToken)
}
