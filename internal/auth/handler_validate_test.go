package auth

import (
	"context"
	"testing"

	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHandler_ValidateToken_Success(t *testing.T) {
	mockService := new(MockService)
	handler := NewHandler(mockService)

	mockService.
		On("Validate", mock.Anything, "validToken").
		Return(&JwtClaims{Id: 1, Role: "admin", UniversityID: 10}, nil)

	req := &authv1.ValidateTokenRequest{Token: "validToken"}

	resp, err := handler.ValidateToken(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, 200, int(resp.Status))
	require.Equal(t, int64(1), resp.UserId)
}
