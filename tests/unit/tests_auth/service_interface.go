package tests_auth

import (
	"context"

	"github.com/lucky720s/diplomaflow/internal/auth"
)

type AuthService interface {
	Register(ctx context.Context, email, password, firstName, lastName, role string, universityID, departmentID int64) (int64, error)
	Login(ctx context.Context, email, password, userAgent, ip string) (string, string, error)
	RefreshToken(ctx context.Context, clientToken, userAgent, ip string) (string, string, error)
	Validate(ctx context.Context, token string) (*auth.JwtClaims, error)
	ListUsers(ctx context.Context, universityID int64, role string, page, pageSize int32, excludeUserID int64) ([]*auth.User, int64, error)
	ListSessions(ctx context.Context, userID int64) ([]*auth.RefreshToken, error)
	RevokeSession(ctx context.Context, userID int64, sessionID uint64) error
}
