package tests_auth

import (
	"context"
	"testing"

	"github.com/lucky720s/diplomaflow/internal/auth"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandler_ValidateToken_Success(t *testing.T) {
	mockService := new(MockService)
	handler := auth.NewHandler(mockService)

	mockService.
		On("Validate", mock.Anything, "validToken").
		Return(&auth.JwtClaims{Id: 1, Role: "admin", UniversityID: 10}, nil)

	req := &authv1.ValidateTokenRequest{Token: "validToken"}

	resp, err := handler.ValidateToken(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, 200, int(resp.Status))
	require.Equal(t, int64(1), resp.UserId)
}
