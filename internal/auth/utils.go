package auth

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type JwtWrapper struct {
	SecretKey       string
	Issuer          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type JwtClaims struct {
	Id           int64
	Email        string
	Role         string   // base role: student|teacher|admin
	DeptRoles    []string // dynamic department role slugs: ["commission","tech_support",...]
	FirstName    string
	LastName     string
	UniversityID int64
	DepartmentID int64
	jwt.RegisteredClaims
}

func (j *JwtWrapper) ValidateToken(signedToken string) (*JwtClaims, error) {
	if strings.TrimSpace(signedToken) == "" {
		return nil, errors.New("token is empty")
	}

	token, err := jwt.ParseWithClaims(
		signedToken,
		&JwtClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(j.SecretKey), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JwtClaims)
	if !ok {
		return nil, errors.New("couldn't parse claims")
	}

	if j.Issuer != "" && claims.Issuer != j.Issuer {
		return nil, fmt.Errorf("invalid issuer: %s", claims.Issuer)
	}
	if claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, errors.New("token expired")
	}

	return claims, nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (j *JwtWrapper) GenerateAccessToken(user User, deptRoles []string) (string, error) {
	claims := &JwtClaims{
		Id:           user.ID,
		Email:        user.Email,
		Role:         user.Role,
		DeptRoles:    deptRoles,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		UniversityID: user.UniversityID,
		DepartmentID: user.DepartmentID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.AccessTokenTTL)),
			Issuer:    j.Issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.SecretKey))
}

func (j *JwtWrapper) GenerateRefreshToken() string { return uuid.New().String() }
func (j *JwtWrapper) GenerateRefreshTokenSecret() string {
	return uuid.New().String()
}

func HashToken(token string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckTokenHash(token, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(token))
	return err == nil
}

func FormatRefreshToken(id uint64, uuidSecret string) string {
	return fmt.Sprintf("%d.%s", id, uuidSecret)
}

func ParseRefreshToken(raw string) (uint64, string, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 2 {
		return 0, "", errors.New("invalid token format")
	}
	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, "", errors.New("invalid token id")
	}
	return id, parts[1], nil
}
