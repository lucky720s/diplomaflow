package auth

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lucky720s/diplomaflow/internal/domain"
	"time"
)

var jwtKey = []byte("supersecretkey")

type Claims struct {
	UserID string      `json:"user_id"`
	Role   domain.Role `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(user *domain.User) (string, error) {
	claims := &Claims{
		UserID: user.ID.String(),
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func ParseToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
