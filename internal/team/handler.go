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
			return nil, status.Errorf(codes.NotFound, "team %s not found", req.GetTeamId())
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
func (h *Handler) ListTeams(ctx context.Context, req *teamv1.ListTeamsRequest) (*teamv1.ListTeamsResponse, error) {
	departmentID, err := getDepartmentIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	teams, err := h.repo.ListTeams(ctx, departmentID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "teams not found")
	}
	resTeams := make([]*teamv1.Team, 0, len(teams))
	for _, team := range teams {
		_, memberIDs, _ := h.repo.GetTeamByID(ctx, team.ID)
		resTeams = append(resTeams, &teamv1.Team{
			Id:           team.ID,
			Name:         team.Name,
			DepartmentId: team.DepartmentID,
			MemberIds:    memberIDs,
		})
	}
	return &teamv1.ListTeamsResponse{Teams: resTeams}, nil
}
func (h *Handler) UpdateTeam(ctx context.Context, req *teamv1.UpdateTeamRequest) (*teamv1.UpdateTeamResponse, error) {
	departmentID, err := getDepartmentIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	team, _, err := h.repo.GetTeamByID(ctx, req.GetTeam().GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "team not found")
	}
	if team.DepartmentID != departmentID {
		return nil, status.Errorf(codes.PermissionDenied, "department_id is not match")
	}
	updatedTeam, err := h.repo.UpdateTeam(ctx, req.GetTeam(), req.GetUpdateMask())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "team not found")
	}
	_, memberIDs, _ := h.repo.GetTeamByID(ctx, updatedTeam.ID)
	return &teamv1.UpdateTeamResponse{
		Team: &teamv1.Team{
			Id:           updatedTeam.ID,
			Name:         updatedTeam.Name,
			DepartmentId: updatedTeam.DepartmentID,
			MemberIds:    memberIDs}}, nil
}
func (h *Handler) DeleteTeam(ctx context.Context, req *teamv1.DeleteTeamRequest) (*teamv1.DeleteTeamResponse, error) {
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
	if err := h.repo.DeleteTeam(ctx, req.GetTeamId()); err != nil {
		return nil, status.Errorf(codes.NotFound, "team not found")
	}
	return &teamv1.DeleteTeamResponse{Success: true}, nil
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
	if err := h.repo.AddMember(ctx, req.GetTeamId(), req.GetUserId()); err != nil {
		return nil, status.Errorf(codes.NotFound, "team not found")
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
