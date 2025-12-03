package role

import (
	"context"

	rolev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/role/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Handler struct {
	rolev1.UnimplementedRoleServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateRole(ctx context.Context, req *rolev1.CreateRoleRequest) (*rolev1.CreateRoleResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "role name is required")
	}

	role, err := h.service.CreateRole(ctx, req.Name, req.DepartmentId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create role: %v", err)
	}

	return &rolev1.CreateRoleResponse{
		Role: &rolev1.Role{
			Id:           role.ID,
			Name:         role.Name,
			DepartmentId: role.DepartmentID,
		},
	}, nil
}

func (h *Handler) DeleteRole(ctx context.Context, req *rolev1.DeleteRoleRequest) (*emptypb.Empty, error) {
	if req.RoleId == 0 {
		return nil, status.Error(codes.InvalidArgument, "role_id is required")
	}

	if err := h.service.DeleteRole(ctx, req.RoleId); err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to delete role: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) GetRole(ctx context.Context, req *rolev1.GetRoleRequest) (*rolev1.GetRoleResponse, error) {
	role, err := h.service.GetRole(ctx, req.RoleId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "role not found")
	}
	return &rolev1.GetRoleResponse{
		Role: &rolev1.Role{
			Id:           role.ID,
			Name:         role.Name,
			DepartmentId: role.DepartmentID,
		},
	}, nil
}
