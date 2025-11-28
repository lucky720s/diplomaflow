package team

import (
	"context"
	"errors"

	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type Handler struct {
	teamv1.UnimplementedTeamServiceServer
	repo       Repository
	authClient authv1.AuthServiceClient
}

func NewHandler(repo Repository, authClient authv1.AuthServiceClient) *Handler {
	return &Handler{repo: repo,
		authClient: authClient}
}

func (h *Handler) CreateTeam(ctx context.Context, req *teamv1.CreateTeamRequest) (*teamv1.CreateTeamResponse, error) {
	departmentID := req.GetDepartmentId()
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
	team, memberIDs, err := h.repo.GetTeamByID(ctx, req.GetTeamId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "team %s not found", req.GetTeamId())
		}
		return nil, status.Errorf(codes.NotFound, "team not found")
	}

	return &teamv1.GetTeamResponse{
		Team: &teamv1.Team{
			Id:           team.ID,
			Name:         team.Name,
			DepartmentId: team.DepartmentID,
			MemberIds:    memberIDs}}, nil
}
func (h *Handler) ListTeams(ctx context.Context, req *teamv1.ListTeamsRequest) (*teamv1.ListTeamsResponse, error) {
	departmentID := req.GetDepartmentId()
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
	_, _, err := h.repo.GetTeamByID(ctx, req.GetTeam().GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "team not found")
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
	_, _, err := h.repo.GetTeamByID(ctx, req.GetTeamId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "team not found")
	}
	if err := h.repo.DeleteTeam(ctx, req.GetTeamId()); err != nil {
		return nil, status.Errorf(codes.NotFound, "team not found")
	}
	return &teamv1.DeleteTeamResponse{Success: true}, nil
}
func (h *Handler) AddMember(ctx context.Context, req *teamv1.AddMemberRequest) (*teamv1.AddMemberResponse, error) {
	_, _, err := h.repo.GetTeamByID(ctx, req.GetTeamId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "team not found")
	}

	if err := h.repo.AddMember(ctx, req.GetTeamId(), req.GetUserId()); err != nil {
		return nil, status.Errorf(codes.NotFound, "team not found")
	}
	return &teamv1.AddMemberResponse{Success: true}, nil
}
func (h *Handler) RemoveMember(ctx context.Context, req *teamv1.RemoveMemberRequest) (*teamv1.RemoveMemberResponse, error) {
	_, _, err := h.repo.GetTeamByID(ctx, req.GetTeamId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "team not found")
	}

	if err := h.repo.RemoveMember(ctx, req.GetTeamId(), req.GetUserId()); err != nil {
		return nil, status.Errorf(codes.Internal, "remove member failed")
	}
	return &teamv1.RemoveMemberResponse{Success: true}, nil
}

func (h *Handler) ListAvailableUsers(ctx context.Context, req *teamv1.ListAvailableUsersRequest) (*teamv1.ListAvailableUsersResponse, error) {
	departmentId := req.GetDepartmentId()
	if departmentId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "department_id is required")
	}
	authRes, err := h.authClient.ListUsersByDepartment(ctx, &authv1.ListUsersByDepartmentRequest{
		DepartmentId: departmentId,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list users failed")
	}
	var availableUsers []*teamv1.UserInfo
	for _, user := range authRes.GetUsers() {
		availableUsers = append(availableUsers, &teamv1.UserInfo{
			Id:    user.GetId(),
			Email: user.GetEmail(),
		})
	}
	return &teamv1.ListAvailableUsersResponse{Users: availableUsers}, nil
}
