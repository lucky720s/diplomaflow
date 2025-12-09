package auth

import "context"

type AuthService interface {
	Register(ctx context.Context, email, password, firstName, lastName, role string, universityID, departmentID int64) (int64, error)
	Login(ctx context.Context, email, password, userAgent, ip string) (string, string, error)
	RefreshToken(ctx context.Context, clientToken, userAgent, ip string) (string, string, error)
	Validate(ctx context.Context, token string) (*JwtClaims, error)
	ListUsers(ctx context.Context, universityID int64, role string, page, pageSize int32, excludeUserID int64) ([]*User, int64, error)
	ListSessions(ctx context.Context, userID int64) ([]*RefreshToken, error)
	RevokeSession(ctx context.Context, userID int64, sessionID uint64) error
}
