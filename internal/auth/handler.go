package auth

import (
	"context"
	"net/http"

	pb "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	pb.UnimplementedAuthServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if req.Email == "" || req.Password == "" {
		return &pb.RegisterResponse{
			Status: http.StatusBadRequest,
			Error:  "email and password are required",
		}, nil
	}

	userID, err := h.service.Register(ctx, req.Email, req.Password, req.FirstName, req.LastName, req.Role, req.UniversityId)
	if err != nil {
		return &pb.RegisterResponse{
			Status: http.StatusConflict,
			Error:  err.Error(),
		}, nil
	}

	return &pb.RegisterResponse{
		Status: http.StatusCreated,
		UserId: userID,
	}, nil
}

func (h *Handler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return &pb.LoginResponse{
			Status: http.StatusBadRequest,
			Error:  "email and password are required",
		}, nil
	}

	token, err := h.service.Login(ctx, req.Email, req.Password)
	if err != nil {
		return &pb.LoginResponse{
			Status: http.StatusUnauthorized,
			Error:  err.Error(),
		}, nil
	}

	return &pb.LoginResponse{
		Status: http.StatusOK,
		Token:  token,
	}, nil
}

func (h *Handler) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	if req.Token == "" {
		return &pb.ValidateTokenResponse{
			Status: http.StatusBadRequest,
			Error:  "token is required",
		}, nil
	}

	claims, err := h.service.Validate(ctx, req.Token)
	if err != nil {
		return &pb.ValidateTokenResponse{
			Status: http.StatusUnauthorized,
			Error:  err.Error(),
		}, nil
	}

	return &pb.ValidateTokenResponse{
		Status:       http.StatusOK,
		UserId:       claims.Id,
		Role:         claims.Role,
		UniversityId: claims.UniversityID,
	}, nil
}

func (h *Handler) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	users, total, err := h.service.ListUsers(ctx, req.UniversityId, req.Role, req.Page, req.PageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}

	var pbUsers []*pb.UserPreview
	for _, u := range users {
		pbUsers = append(pbUsers, &pb.UserPreview{
			Id:           u.ID,
			Email:        u.Email,
			FirstName:    u.FirstName,
			LastName:     u.LastName,
			Role:         u.Role,
			UniversityId: u.UniversityID,
		})
	}

	return &pb.ListUsersResponse{
		Users:      pbUsers,
		TotalCount: total,
	}, nil
}
