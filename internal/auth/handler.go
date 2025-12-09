package auth

import (
	"context"
	"net/http"

	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	authv1.UnimplementedAuthServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}
	if err := req.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation error: %v", err)
	}
	userID, err := h.service.Register(ctx, req.Email, req.Password, req.FirstName, req.LastName, req.Role, req.UniversityId, req.DepartmentId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register: %v", err)
	}
	return &authv1.RegisterResponse{
		UserId: userID,
	}, nil
}

func (h *Handler) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	token, err := h.service.Login(ctx, req.Email, req.Password)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	return &authv1.LoginResponse{
		Token: token,
	}, nil
}

func (h *Handler) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	if req.Token == "" {
		return &authv1.ValidateTokenResponse{
			Status: http.StatusBadRequest,
			Error:  "token is required",
		}, nil
	}

	claims, err := h.service.Validate(ctx, req.Token)
	if err != nil {
		return &authv1.ValidateTokenResponse{
			Status: http.StatusUnauthorized,
			Error:  err.Error(),
		}, nil
	}

	return &authv1.ValidateTokenResponse{
		Status:       http.StatusOK,
		UserId:       claims.Id,
		Role:         claims.Role,
		UniversityId: claims.UniversityID,
		DepartmentId: claims.DepartmentID,
	}, nil
}

func (h *Handler) ListUsers(ctx context.Context, req *authv1.ListUsersRequest) (*authv1.ListUsersResponse, error) {
	users, total, err := h.service.ListUsers(ctx, req.UniversityId, req.Role, req.Page, req.PageSize, req.ExcludeUserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}

	var pbUsers []*authv1.UserPreview
	for _, u := range users {
		pbUsers = append(pbUsers, &authv1.UserPreview{
			Id:           u.ID,
			Email:        u.Email,
			FirstName:    u.FirstName,
			LastName:     u.LastName,
			Role:         u.Role,
			UniversityId: u.UniversityID,
		})
	}

	return &authv1.ListUsersResponse{
		Users:      pbUsers,
		TotalCount: total,
	}, nil
}
