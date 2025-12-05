package auth

import "context"

type AuthService interface {
	Register(ctx context.Context, email, password, firstName, lastName, role string, universityID int64) (int64, error)
	Login(ctx context.Context, email, password string) (string, error)
	Validate(ctx context.Context, token string) (*JwtClaims, error)
	ListUsers(ctx context.Context, universityID int64, role string, page, pageSize int32) ([]*User, int64, error)
}
