package team

import (
	"context"
	"errors"
	"strconv"

	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type Handler struct {
	teamv1.UnimplementedTeamServiceServer
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}
func getDepartmentIDFromContext(ctx context.Context) (int64, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		status.Error(codes.Unauthenticated, "metadata is not provided")

	}
	values := md.Get("department_id")
	if len(values) == 0 {
		return 0, status.Error(codes.Unauthenticated, "department_id is not provided")
	}
	departmentID, err := strconv.ParseInt(values[0], 10, 64)
	if err != nil {
		return 0, status.Error(codes.Unauthenticated, "department_id is not provided")
	}
	return departmentID, nil
}

func (h *Handler) CreateTeam(ctx context.Context, req *teamv1.CreateTeamRequest) (*teamv1.CreateTeamResponse, error) {
	departmentID, err := getDepartmentIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	team, err := h.repo.CreateTeam(ctx, req.GetName(), departmentID, req.GetMemberIds())
	if err != nil {
		return nil, err
	}
	return &teamv1.CreateTeamResponse{
		Team: &teamv1.Team{
			Id:           team.ID,
			Name:         team.Name,
			DepartmentId: team.DepartmentID,
			MemberIds:    req.GetMemberIds(),
		}}, nil
}
func (h *Handler) GetTeam(ctx context.Context, req *teamv1.GetTeamRequest) (*teamv1.GetTeamResponse, error) {
	departmentID, err := getDepartmentIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	team, memberIDs, err := h.repo.GetTeamByID(ctx, req.GetTeamId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, status.Errorf(codes.NotFound, "team not found")
	}
	if team.DepartmentID != departmentID {
		return nil, status.Errorf(codes.PermissionDenied, "department_id is not match")
	}
	return &teamv1.GetTeamResponse{
		Team: &teamv1.Team{
			Id:           team.ID,
			Name:         team.Name,
			DepartmentId: team.DepartmentID,
			MemberIds:    memberIDs}}, nil
}
func (h *Handler) AddMember(ctx context.Context, req *teamv1.AddMemberRequest) (*teamv1.AddMemberResponse, error) {
	departmentID, err := getDepartmentIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	team, _, err := h.repo.GetTeamByID(ctx, req.GetTeamId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "team not found")
	}
	if team.DepartmentID != departmentID {
		return nil, status.Errorf(codes.PermissionDenied, "department_id is not match")
	}
	return &teamv1.AddMemberResponse{Success: true}, nil
}
func (h *Handler) RemoveMember(ctx context.Context, req *teamv1.RemoveMemberRequest) (*teamv1.RemoveMemberResponse, error) {
	departmentID, err := getDepartmentIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	team, _, err := h.repo.GetTeamByID(ctx, req.GetTeamId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "team not found")
	}
	if team.DepartmentID != departmentID {
		return nil, status.Errorf(codes.PermissionDenied, "department_id is not match")
	}
	if err := h.repo.RemoveMember(ctx, req.GetTeamId(), req.GetUserId()); err != nil {
		return nil, status.Errorf(codes.Internal, "remove member failed")
	}
	return &teamv1.RemoveMemberResponse{Success: true}, nil
}
