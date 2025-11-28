package auth

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	rolev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/role/v1"
	universityv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/university/v1"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type Handler struct {
	authv1.UnimplementedAuthServiceServer
	repo             Repository
	universityClient universityv1.UniversityServiceClient
	roleClient       rolev1.RoleServiceClient
}

func NewHandler(repo Repository, universityClient universityv1.UniversityServiceClient, roleClient rolev1.RoleServiceClient) *Handler {
	return &Handler{
		repo:             repo,
		universityClient: universityClient,
		roleClient:       roleClient,
	}
}

func (h *Handler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	user, err := h.repo.CreateUser(ctx, req.GetEmail(), req.GetPassword(), req.GetUniversityId(), req.GetDepartmentId())
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "User does not exist")
	}
	return &authv1.RegisterResponse{UserId: user.ID}, nil
}

func (h *Handler) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	user, err := h.repo.GetUserByEmail(ctx, req.GetEmail())
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "failed to login")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.GetPassword())); err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid password")
	}
	rolesIDs, err := h.repo.GetUserRoleIDs(ctx, user.ID)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "role id does not exist")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   user.ID,
		"uid":   user.UniversityID,
		"did":   user.DepartmentID,
		"roles": rolesIDs,
		"exp":   time.Now().Add(time.Hour * 24).Unix(),
	})
	jwtSecret := os.Getenv("JWT_SECRET")
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to sign token")
	}
	return &authv1.LoginResponse{Token: tokenString}, nil
}

func (h *Handler) GetUser(ctx context.Context, req *authv1.GetUserRequest) (*authv1.GetUserResponse, error) {
	user, err := h.repo.GetUserByID(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.Internal, "user id does not exist")
		}
	}
	roleIDs, err := h.repo.GetUserRoleIDs(ctx, user.ID)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "role id does not exist")
	}
	var roles []*rolev1.Role
	for _, roleID := range roleIDs {
		roleRes, err := h.roleClient.GetRole(ctx, &rolev1.GetRoleRequest{RoleId: roleID})
		if err == nil {
			roles = append(roles, roleRes.GetRole())
		}
	}
	return &authv1.GetUserResponse{
		Id:           user.ID,
		Email:        user.Email,
		DepartmentId: user.DepartmentID,
		Roles:        roles}, nil
}
func (h *Handler) AssignRole(ctx context.Context, req *authv1.AssignRoleRequest) (*authv1.AssignRoleResponse, error) {
	if err := h.repo.AssignRole(ctx, req.GetUserId(), req.GetRoleId()); err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "failed to assign role")
	}
	return &authv1.AssignRoleResponse{Success: true}, nil
}
func (h *Handler) ListUsersByDepartment(ctx context.Context, req *authv1.ListUsersByDepartmentRequest) (*authv1.ListUsersByDepartmentResponse, error) {
	users, err := h.repo.GetUsersByDepartment(ctx, req.GetDepartmentId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list users")
	}
	var pbUsers []*authv1.UserInfo
	for _, user := range users {
		pbUsers = append(pbUsers, &authv1.UserInfo{
			Id:    user.ID,
			Email: user.Email,
		})
	}
	return &authv1.ListUsersByDepartmentResponse{Users: pbUsers}, nil
}
