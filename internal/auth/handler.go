package auth

import (
	"context"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	auth_pb "github.com/lucky720s/diplomaflow/pkg/protobuf/auth"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"os"
	"time"
)

type Handler struct {
	auth_pb.UnimplementedAuthServiceServer
	repo *AuthRepository
}

func NewHandler() *Handler {
	return &Handler{repo: NewAuthRepository()}
}
func (h *Handler) Register(ctx context.Context, req *auth_pb.RegisterRequest) (*auth_pb.RegisterResponse, error) {
	user, err := h.repo.CreateUser(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, err
	}
	return &auth_pb.RegisterResponse{UserId: user.ID}, nil
}

func (h *Handler) Login(ctx context.Context, req *auth_pb.LoginRequest) (*auth_pb.LoginResponse, error) {
	user, err := h.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash),
		[]byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})
	jwtSecret := os.Getenv("JWT_SECRET")
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return nil, err
	}
	return &auth_pb.LoginResponse{Token: tokenString}, nil
}
