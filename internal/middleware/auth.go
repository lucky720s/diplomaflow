package middleware

import (
	"context"
	"github.com/google/uuid"
	"github.com/lucky720s/diplomaflow/internal/auth"
	"github.com/lucky720s/diplomaflow/internal/domain"
	"net/http"
	"strings"
)

type contextKey string

const UserContextKey contextKey = "user"

type UserContext struct {
	ID   uuid.UUID
	Role domain.Role
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}

		tokenParts := strings.Split(header, "Bearer ")
		if len(tokenParts) != 2 {
			http.Error(w, "Invalid token format", http.StatusUnauthorized)
			return
		}

		claims, err := auth.ParseToken(tokenParts[1])
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			http.Error(w, "Invalid user ID in token", http.StatusUnauthorized)
			return
		}

		userCtx := &UserContext{
			ID:   userID,
			Role: claims.Role,
		}

		ctx := context.WithValue(r.Context(), UserContextKey, userCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RoleMiddleware(allowedRoles ...domain.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := r.Context().Value(UserContextKey).(*UserContext)
			if !ok || user == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			for _, role := range allowedRoles {
				if user.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, "Forbidden", http.StatusForbidden)
		})
	}
}
