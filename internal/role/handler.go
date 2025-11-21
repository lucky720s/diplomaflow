package role

import (
	"context"
	"errors"
	"strconv"

	rolev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/role/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type Handler struct {
	rolev1.UnimplementedRoleServiceServer
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}
func getDepartmentIDFromContext(ctx context.Context) (int64, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, status.Error(codes.FailedPrecondition, "Failed to get metadata from context")
	}
	values := md.Get("department-id")
	if len(values) == 0 {
		return 0, status.Error(codes.FailedPrecondition, "Failed to get department id")
	}
	departmentID, err := strconv.ParseInt(values[0], 10, 64)
	if err != nil {
		return 0, status.Error(codes.FailedPrecondition, "Failed to get department id")
	}
	return departmentID, nil
}
func (h *Handler) CreateRole(ctx context.Context, req *rolev1.CreateRoleRequest) (*rolev1.CreateRoleResponse, error) {
	departmentID, err := getDepartmentIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	role, err := h.repo.CreateRole(ctx, req.GetName(), departmentID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to create role: %v", err)
	}
	return &rolev1.CreateRoleResponse{
		Role: &rolev1.Role{
			Id:           role.ID,
			Name:         role.Name,
			DepartmentId: role.DepartmentID,
		}}, nil
}
func (h *Handler) GetRole(ctx context.Context, req *rolev1.GetRoleRequest) (*rolev1.GetRoleResponse, error) {
	departmentID, err := getDepartmentIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	role, err := h.repo.GetRole(ctx, req.GetRoleId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "role not found")
		}
		return nil, status.Errorf(codes.Internal, "Failed to get role: %v", err)
	}
	if role.DepartmentID != departmentID {
		return nil, status.Errorf(codes.FailedPrecondition, "role department id does not match")
	}
	return &rolev1.GetRoleResponse{
		Role: &rolev1.Role{
			Id:           role.ID,
			Name:         role.Name,
			DepartmentId: role.DepartmentID,
		}}, nil
}
func (h *Handler) ListRoles(ctx context.Context, req *rolev1.ListRolesRequest) (*rolev1.ListRolesResponse, error) {
	departmentID, err := getDepartmentIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	roles, err := h.repo.ListRoles(ctx, departmentID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to list roles: %v", err)
	}
	resRoles := make([]*rolev1.Role, len(roles))
	for i, role := range roles {
		resRoles[i] = &rolev1.Role{
			Id:           role.ID,
			Name:         role.Name,
			DepartmentId: role.DepartmentID,
		}
	}
	return &rolev1.ListRolesResponse{
		Roles: resRoles}, nil
}
func (h *Handler) DeleteRole(ctx context.Context, req *rolev1.DeleteRoleRequest) (*rolev1.DeleteRoleResponse, error) {
	departmentID, err := getDepartmentIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	role, err := h.repo.GetRole(ctx, req.GetRoleId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
		}
	}
	if role.DepartmentID != departmentID {
		return nil, status.Errorf(codes.FailedPrecondition, "role department id does not match")
	}
	if err := h.repo.DeleteRole(ctx, req.GetRoleId()); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to delete role: %v", err)
	}
	return &rolev1.DeleteRoleResponse{Success: true}, nil

}
