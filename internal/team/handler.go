package team

import (
	"context"
	"strconv"

	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Handler struct {
	teamv1.UnimplementedTeamServiceServer
	service *Service
	logger  *zap.Logger
}

func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// getRequesterID извлекает user_id из gRPC metadata
func getRequesterID(ctx context.Context) int64 {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("x-user-id"); len(vals) > 0 {
			id, _ := strconv.ParseInt(vals[0], 10, 64)
			return id
		}
	}
	return 0
}

func (h *Handler) CreateTeam(ctx context.Context, req *teamv1.CreateTeamRequest) (*teamv1.CreateTeamResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.LeaderId == 0 {
		return nil, status.Error(codes.InvalidArgument, "leader_id is required")
	}

	teamID, err := h.service.CreateTeam(ctx, req.Name, req.LeaderId, req.MemberIds, 0)
	if err != nil {
		h.logger.Error("CreateTeam failed", zap.Error(err))
		return nil, mapError(err)
	}

	return &teamv1.CreateTeamResponse{TeamId: teamID}, nil
}

func (h *Handler) GetTeam(ctx context.Context, req *teamv1.GetTeamRequest) (*teamv1.GetTeamResponse, error) {
	team, members, err := h.service.GetTeam(ctx, req.TeamId)
	if err != nil {
		return nil, mapError(err)
	}

	var pbMembers []*teamv1.TeamMember
	for _, m := range members {
		pbMembers = append(pbMembers, &teamv1.TeamMember{
			UserId: m.UserID,
			Role:   m.Role,
		})
	}

	return &teamv1.GetTeamResponse{
		TeamId:    team.ID,
		Name:      team.Name,
		ProjectId: team.ProjectID,
		Members:   pbMembers,
	}, nil
}

func (h *Handler) GetMyTeam(ctx context.Context, req *teamv1.GetMyTeamRequest) (*teamv1.GetMyTeamResponse, error) {
	team, membership, members, pendingCount, err := h.service.GetMyTeam(ctx, req.UserId)
	if err != nil {
		return nil, mapError(err)
	}

	if team == nil {
		return &teamv1.GetMyTeamResponse{HasTeam: false}, nil
	}

	var pbMembers []*teamv1.TeamMember
	for _, m := range members {
		pbMembers = append(pbMembers, &teamv1.TeamMember{
			UserId: m.UserID,
			Role:   m.Role,
		})
	}

	return &teamv1.GetMyTeamResponse{
		HasTeam: true,
		Team: &teamv1.TeamInfo{
			TeamId:              team.ID,
			Name:                team.Name,
			ProjectId:           team.ProjectID,
			Role:                membership.Role,
			Members:             pbMembers,
			MemberCount:         int32(len(members)),
			PendingInvitesCount: int32(pendingCount),
		},
	}, nil
}

func (h *Handler) UpdateTeam(ctx context.Context, req *teamv1.UpdateTeamRequest) (*teamv1.UpdateTeamResponse, error) {
	if req.Team == nil {
		return nil, status.Error(codes.InvalidArgument, "team is required")
	}

	// Получаем requester_id из metadata
	requesterID := getRequesterID(ctx)
	if requesterID == 0 {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	team, err := h.service.UpdateTeam(ctx, req.Team.Id, req.Team.Name, requesterID)
	if err != nil {
		h.logger.Error("UpdateTeam failed", zap.Error(err))
		return nil, mapError(err)
	}

	members, _ := h.service.repo.GetMembers(ctx, team.ID)
	var pbMembers []*teamv1.TeamMember
	for _, m := range members {
		pbMembers = append(pbMembers, &teamv1.TeamMember{
			UserId: m.UserID,
			Role:   m.Role,
		})
	}

	return &teamv1.UpdateTeamResponse{
		Team: &teamv1.Team{
			Id:        team.ID,
			Name:      team.Name,
			ProjectId: team.ProjectID,
			Members:   pbMembers,
			CreatedAt: team.CreatedAt.Format("2006-01-02T15:04:05Z"),
		},
	}, nil
}

func (h *Handler) DeleteTeam(ctx context.Context, req *teamv1.DeleteTeamRequest) (*emptypb.Empty, error) {
	// Получаем requester_id из metadata
	requesterID := getRequesterID(ctx)
	if requesterID == 0 {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	if err := h.service.DeleteTeam(ctx, req.TeamId, requesterID); err != nil {
		h.logger.Error("DeleteTeam failed", zap.Error(err))
		return nil, mapError(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) LeaveTeam(ctx context.Context, req *teamv1.LeaveTeamRequest) (*teamv1.LeaveTeamResponse, error) {
	if req.TeamId == 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id is required")
	}
	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	result, err := h.service.LeaveTeam(ctx, req.TeamId, req.UserId)
	if err != nil {
		h.logger.Error("LeaveTeam failed", zap.Error(err))
		return nil, mapError(err)
	}

	return &teamv1.LeaveTeamResponse{
		Success:     result.Success,
		Message:     result.Message,
		TeamDeleted: result.TeamDeleted,
		NewLeaderId: result.NewLeaderID,
	}, nil
}

func (h *Handler) TransferLeadership(ctx context.Context, req *teamv1.TransferLeadershipRequest) (*teamv1.TransferLeadershipResponse, error) {
	if req.TeamId == 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id is required")
	}
	if req.CurrentLeaderId == 0 {
		return nil, status.Error(codes.InvalidArgument, "current_leader_id is required")
	}
	if req.NewLeaderId == 0 {
		return nil, status.Error(codes.InvalidArgument, "new_leader_id is required")
	}

	team, members, err := h.service.TransferLeadership(ctx, req.TeamId, req.CurrentLeaderId, req.NewLeaderId)
	if err != nil {
		h.logger.Error("TransferLeadership failed", zap.Error(err))
		return nil, mapError(err)
	}

	var pbMembers []*teamv1.TeamMember
	for _, m := range members {
		pbMembers = append(pbMembers, &teamv1.TeamMember{
			UserId: m.UserID,
			Role:   m.Role,
		})
	}

	return &teamv1.TransferLeadershipResponse{
		Success: true,
		Message: "Лидерство успешно передано",
		UpdatedTeam: &teamv1.TeamInfo{
			TeamId:      team.ID,
			Name:        team.Name,
			ProjectId:   team.ProjectID,
			Members:     pbMembers,
			MemberCount: int32(len(members)),
		},
	}, nil
}

func (h *Handler) AddMember(ctx context.Context, req *teamv1.AddMemberRequest) (*teamv1.AddMemberResponse, error) {
	if err := h.service.AddMember(ctx, req.TeamId, req.UserId, req.Role); err != nil {
		return nil, mapError(err)
	}
	return &teamv1.AddMemberResponse{
		Success: true,
		Message: "Member added successfully",
	}, nil
}

func (h *Handler) RemoveMember(ctx context.Context, req *teamv1.RemoveMemberRequest) (*emptypb.Empty, error) {
	// Получаем requester_id из metadata
	requesterID := getRequesterID(ctx)
	if requesterID == 0 {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	if err := h.service.RemoveMember(ctx, req.TeamId, req.UserId, requesterID); err != nil {
		return nil, mapError(err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) GetMyInvites(ctx context.Context, req *teamv1.GetMyInvitesRequest) (*teamv1.GetMyInvitesResponse, error) {
	invites, err := h.service.GetMyInvites(ctx, req.UserId)
	if err != nil {
		return nil, mapError(err)
	}

	var pbInvites []*teamv1.Invite
	for _, inv := range invites {
		team, _ := h.service.repo.GetByID(ctx, inv.TeamID)
		teamName := ""
		if team != nil {
			teamName = team.Name
		}

		pbInvites = append(pbInvites, &teamv1.Invite{
			Id:        inv.ID,
			TeamId:    inv.TeamID,
			TeamName:  teamName,
			InviterId: inv.InviterID,
			Status:    inv.Status,
		})
	}

	return &teamv1.GetMyInvitesResponse{Invites: pbInvites}, nil
}

func (h *Handler) RespondToInvite(ctx context.Context, req *teamv1.RespondToInviteRequest) (*teamv1.RespondToInviteResponse, error) {
	if err := h.service.RespondToInvite(ctx, req.InviteId, req.UserId, req.Accept); err != nil {
		return nil, mapError(err)
	}

	message := "Приглашение отклонено"
	if req.Accept {
		message = "Вы присоединились к команде"
	}

	return &teamv1.RespondToInviteResponse{
		Success: true,
		Message: message,
	}, nil
}

func (h *Handler) GetAvailableStudents(ctx context.Context, req *teamv1.GetAvailableStudentsRequest) (*teamv1.GetAvailableStudentsResponse, error) {
	users, err := h.service.GetAvailableStudents(ctx, req.UniversityId, req.GetDepartmentId(), req.ExcludeUserId)
	if err != nil {
		return nil, mapError(err)
	}

	var students []*teamv1.StudentPreview
	for _, u := range users {
		students = append(students, &teamv1.StudentPreview{
			Id:       u.Id,
			FullName: u.FirstName + " " + u.LastName,
			Email:    u.Email,
		})
	}

	return &teamv1.GetAvailableStudentsResponse{Students: students}, nil
}

func (h *Handler) AssignProject(ctx context.Context, req *teamv1.AssignProjectRequest) (*teamv1.AssignProjectResponse, error) {
	if err := h.service.AssignProject(ctx, req.TeamId, req.ProjectId); err != nil {
		return nil, mapError(err)
	}
	return &teamv1.AssignProjectResponse{Success: true}, nil
}

func (h *Handler) ListTeams(ctx context.Context, req *teamv1.ListTeamsRequest) (*teamv1.ListTeamsResponse, error) {
	teams, total, err := h.service.ListTeams(ctx, req.DepartmentId, req.ProjectId, req.Page, req.PageSize)
	if err != nil {
		return nil, mapError(err)
	}

	var pbTeams []*teamv1.Team
	for _, t := range teams {
		members, _ := h.service.repo.GetMembers(ctx, t.ID)
		var pbMembers []*teamv1.TeamMember
		for _, m := range members {
			pbMembers = append(pbMembers, &teamv1.TeamMember{
				UserId: m.UserID,
				Role:   m.Role,
			})
		}

		pbTeams = append(pbTeams, &teamv1.Team{
			Id:        t.ID,
			Name:      t.Name,
			ProjectId: t.ProjectID,
			Members:   pbMembers,
			CreatedAt: t.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	return &teamv1.ListTeamsResponse{
		Teams:      pbTeams,
		TotalCount: total,
	}, nil
}

func mapError(err error) error {
	switch err {
	case ErrTeamNotFound:
		return status.Error(codes.NotFound, "team not found")
	case ErrMemberNotFound:
		return status.Error(codes.NotFound, "member not found")
	case ErrNotTeamMember:
		return status.Error(codes.PermissionDenied, "user is not a team member")
	case ErrNotLeader:
		return status.Error(codes.PermissionDenied, "only team leader can perform this action")
	case ErrAlreadyInTeam:
		return status.Error(codes.AlreadyExists, "user is already in a team")
	case ErrCannotRemoveLeader:
		return status.Error(codes.FailedPrecondition, "cannot remove team leader")
	case ErrNewLeaderNotInTeam:
		return status.Error(codes.InvalidArgument, "new leader must be a team member")
	case ErrUnauthorized:
		return status.Error(codes.PermissionDenied, "unauthorized")
	default:
		return status.Errorf(codes.Internal, "internal error: %v", err)
	}
}
