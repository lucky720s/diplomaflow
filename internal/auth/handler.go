package auth

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	auth_pb "github.com/lucky720s/diplomaflow/pkg/protobuf/auth"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
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

func (h *Handler) GetUser(ctx context.Context, req *auth_pb.GetUserRequest) (*auth_pb.GetUserResponse, error) {
	user, err := h.repo.GetUserByID(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "user with id %d not found", req.GetUserId())
		}
		return nil, status.Errorf(codes.Internal, "error getting user with id %d: %v", req.GetUserId(), err)

	}
	return &auth_pb.GetUserResponse{
		Id:    user.ID,
		Email: user.Email,
	}, nil
}
