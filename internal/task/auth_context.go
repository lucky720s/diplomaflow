package task

import (
	"context"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthContext struct {
	UserID       int64
	Role         string
	UniversityID int64
	DepartmentID int64
}

type authContextKey struct{}

func AuthorizationInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if isHealthCheck(info.FullMethod) {
			return handler(ctx, req)
		}
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		if svc := mdString(md, "x-internal-service"); svc != "" {
			ctx = context.WithValue(ctx, authContextKey{}, AuthContext{
				UserID: -1, // Маркер internal call
				Role:   "internal:" + svc,
			})
			return handler(ctx, req)
		}

		authCtx := AuthContext{
			UserID:       mdInt64(md, "x-user-id"),
			Role:         mdString(md, "x-user-role"),
			UniversityID: mdInt64(md, "x-university-id"),
			DepartmentID: mdInt64(md, "x-department-id"),
		}

		if authCtx.UserID == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing x-user-id")
		}

		if authCtx.Role == "" {
			return nil, status.Error(codes.Unauthenticated, "missing x-user-role")
		}

		ctx = context.WithValue(ctx, authContextKey{}, authCtx)

		return handler(ctx, req)
	}
}

func GetAuthContext(ctx context.Context) (AuthContext, bool) {
	ac, ok := ctx.Value(authContextKey{}).(AuthContext)
	return ac, ok
}

func MustGetAuthContext(ctx context.Context) AuthContext {
	ac, ok := GetAuthContext(ctx)
	if !ok {
		panic("auth context not found - AuthorizationInterceptor not configured?")
	}
	return ac
}

func IsInternalCall(auth AuthContext) bool {
	return auth.UserID == -1
}

func GetInternalServiceName(auth AuthContext) string {
	if !IsInternalCall(auth) {
		return ""
	}
	if len(auth.Role) > 9 {
		return auth.Role[9:]
	}
	return ""
}

func isHealthCheck(method string) bool {
	return method == "/grpc.health.v1.Health/Check" ||
		method == "/grpc.health.v1.Health/Watch"
}

func mdInt64(md metadata.MD, key string) int64 {
	vals := md.Get(key)
	if len(vals) == 0 {
		return 0
	}
	n, err := strconv.ParseInt(vals[0], 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func mdString(md metadata.MD, key string) string {
	vals := md.Get(key)
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}
