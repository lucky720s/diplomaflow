package team

import (
	"context"
	"fmt"

	pb "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type TeamUseCase interface {
	CreateTeam(ctx context.Context, name string, projectID *int64, memberIDs []int64, leaderID int64, departmentID int64) (int64, error)
	// ✅ Изменено: uint64 вместо int64
	GetTeam(ctx context.Context, id uint64) (*Team, error)
	GetAvailableStudents(ctx context.Context, universityID, excludeUserID int64) ([]*StudentPreview, error)
	AssignProject(ctx context.Context, teamID, projectID int64) error
	GetMyInvites(ctx context.Context, userID int64) ([]*TeamInvite, error)
	RespondToInvite(ctx context.Context, inviteID, userID int64, accept bool) error
	GetMyTeam(ctx context.Context, userID int64) (*Team, string, int64, error)
	ListTeams(ctx context.Context, departmentID, projectID int64, page, pageSize int32) ([]*Team, int64, error)
	UpdateTeam(ctx context.Context, teamID uint64, name string) (*Team, error)
	DeleteTeam(ctx context.Context, teamID uint64) error
	AddMember(ctx context.Context, teamID uint64, userID int64, role string) error
	RemoveMember(ctx context.Context, teamID uint64, userID int64) error
}

type Handler struct {
	teamv1.UnimplementedTeamServiceServer
	service TeamUseCase
	log     *zap.Logger
}

func NewHandler(service TeamUseCase, log *zap.Logger) *Handler {
	return &Handler{
		service: service,
		log:     log,
	}
}
func (h *Handler) CreateTeam(ctx context.Context, req *teamv1.CreateTeamRequest) (*teamv1.CreateTeamResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.LeaderId == 0 {
		return nil, status.Error(codes.InvalidArgument, "leader_id is required")
	}

	var projectID *int64
	if req.ProjectId != 0 {
		projectID = &req.ProjectId
	}
	var departmentID int64
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("x-department-id"); len(vals) > 0 {
			_, _ = fmt.Sscanf(vals[0], "%d", &departmentID)
		}
	}
	teamID, err := h.service.CreateTeam(ctx, req.Name, projectID, req.MemberIds, req.LeaderId, departmentID)
	if err != nil {
		h.log.Error("CreateTeam failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create team: %v", err)
	}

	return &teamv1.CreateTeamResponse{TeamId: teamID}, nil
}

func (h *Handler) GetTeam(ctx context.Context, req *pb.GetTeamRequest) (*pb.GetTeamResponse, error) {
	if req.TeamId == 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id is required")
	}

	team, err := h.service.GetTeam(ctx, uint64(req.TeamId))
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "team not found: %v", err)
	}

	var pbMembers []*pb.TeamMember
	for _, m := range team.Members {
		pbMembers = append(pbMembers, &pb.TeamMember{
			UserId: m.UserID,
			Role:   m.Role,
		})
	}
	var projID int64
	if team.ProjectID != nil {
		projID = *team.ProjectID
	}
	return &pb.GetTeamResponse{
		TeamId:    int64(team.ID),
		Name:      team.Name,
		ProjectId: projID,
		Members:   pbMembers,
	}, nil
}

func (h *Handler) GetAvailableStudents(ctx context.Context, req *pb.GetAvailableStudentsRequest) (*pb.GetAvailableStudentsResponse, error) {
	if req.UniversityId == 0 {
		return nil, status.Error(codes.InvalidArgument, "university_id is required")
	}

	students, err := h.service.GetAvailableStudents(ctx, req.UniversityId, req.ExcludeUserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get students: %v", err)
	}

	var pbStudents []*pb.StudentPreview
	for _, s := range students {
		pbStudents = append(pbStudents, &pb.StudentPreview{
			Id:       s.ID,
			FullName: s.FirstName + " " + s.LastName,
			Email:    s.Email,
		})
	}

	return &pb.GetAvailableStudentsResponse{
		Students: pbStudents,
	}, nil
}
func (h *Handler) AssignProject(ctx context.Context, req *pb.AssignProjectRequest) (*pb.AssignProjectResponse, error) {
	if req.TeamId == 0 || req.ProjectId == 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id and project_id are required")
	}

	err := h.service.AssignProject(ctx, req.TeamId, req.ProjectId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to assign project: %v", err)
	}

	return &pb.AssignProjectResponse{Success: true}, nil
}
func (h *Handler) GetMyInvites(ctx context.Context, req *pb.GetMyInvitesRequest) (*pb.GetMyInvitesResponse, error) {
	invites, err := h.service.GetMyInvites(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed: %v", err)
	}
	var pbInvites []*pb.Invite
	for _, inv := range invites {
		pbInvites = append(pbInvites, &pb.Invite{
			Id:        int64(inv.ID),
			TeamId:    int64(inv.TeamID),
			TeamName:  inv.Team.Name,
			InviterId: inv.InviterID,
			Status:    inv.Status,
		})
	}
	return &pb.GetMyInvitesResponse{Invites: pbInvites}, nil
}

func (h *Handler) RespondToInvite(ctx context.Context, req *pb.RespondToInviteRequest) (*pb.RespondToInviteResponse, error) {
	err := h.service.RespondToInvite(ctx, req.InviteId, req.UserId, req.Accept)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed: %v", err)
	}
	return &pb.RespondToInviteResponse{Success: true}, nil
}
func (h *Handler) GetMyTeam(ctx context.Context, req *pb.GetMyTeamRequest) (*pb.GetMyTeamResponse, error) {
	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	team, role, pendingCount, err := h.service.GetMyTeam(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get team: %v", err)
	}
	if team == nil {
		return &pb.GetMyTeamResponse{
			HasTeam: false,
			Team:    nil,
		}, nil
	}
	var pbMembers []*pb.TeamMember
	for _, m := range team.Members {
		pbMembers = append(pbMembers, &pb.TeamMember{
			UserId: m.UserID,
			Role:   m.Role,
		})
	}

	var projID int64
	if team.ProjectID != nil {
		projID = *team.ProjectID
	}

	return &pb.GetMyTeamResponse{
		HasTeam: true,
		Team: &pb.TeamInfo{
			TeamId:              int64(team.ID),
			Name:                team.Name,
			ProjectId:           projID,
			Role:                role,
			Members:             pbMembers,
			MemberCount:         int32(len(team.Members)),
			PendingInvitesCount: int32(pendingCount),
		},
	}, nil
}
func (h *Handler) ListTeams(ctx context.Context, req *pb.ListTeamsRequest) (*pb.ListTeamsResponse, error) {
	teams, total, err := h.service.ListTeams(ctx, req.DepartmentId, req.ProjectId, req.Page, req.PageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list teams: %v", err)
	}

	var pbTeams []*pb.Team
	for _, t := range teams {
		var members []*pb.TeamMember
		for _, m := range t.Members {
			members = append(members, &pb.TeamMember{
				UserId: m.UserID,
				Role:   m.Role,
			})
		}

		var projID int64
		if t.ProjectID != nil {
			projID = *t.ProjectID
		}

		pbTeams = append(pbTeams, &pb.Team{
			Id:        int64(t.ID),
			Name:      t.Name,
			ProjectId: projID,
			Members:   members,
			CreatedAt: t.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	return &pb.ListTeamsResponse{
		Teams:      pbTeams,
		TotalCount: total,
	}, nil
}

func (h *Handler) UpdateTeam(ctx context.Context, req *pb.UpdateTeamRequest) (*pb.UpdateTeamResponse, error) {
	if req.Team == nil || req.Team.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "team with id is required")
	}

	team, err := h.service.UpdateTeam(ctx, uint64(req.Team.Id), req.Team.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update team: %v", err)
	}

	var members []*pb.TeamMember
	for _, m := range team.Members {
		members = append(members, &pb.TeamMember{
			UserId: m.UserID,
			Role:   m.Role,
		})
	}

	var projID int64
	if team.ProjectID != nil {
		projID = *team.ProjectID
	}

	return &pb.UpdateTeamResponse{
		Team: &pb.Team{
			Id:        int64(team.ID),
			Name:      team.Name,
			ProjectId: projID,
			Members:   members,
			CreatedAt: team.CreatedAt.Format("2006-01-02T15:04:05Z"),
		},
	}, nil
}

func (h *Handler) DeleteTeam(ctx context.Context, req *pb.DeleteTeamRequest) (*emptypb.Empty, error) {
	if req.TeamId == 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id is required")
	}

	if err := h.service.DeleteTeam(ctx, uint64(req.TeamId)); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete team: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (h *Handler) AddMember(ctx context.Context, req *pb.AddMemberRequest) (*pb.AddMemberResponse, error) {
	if req.TeamId == 0 || req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id and user_id are required")
	}

	if err := h.service.AddMember(ctx, uint64(req.TeamId), req.UserId, req.Role); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add member: %v", err)
	}

	return &pb.AddMemberResponse{
		Success: true,
		Message: "Member added successfully",
	}, nil
}

func (h *Handler) RemoveMember(ctx context.Context, req *pb.RemoveMemberRequest) (*emptypb.Empty, error) {
	if req.TeamId == 0 || req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "team_id and user_id are required")
	}

	if err := h.service.RemoveMember(ctx, uint64(req.TeamId), req.UserId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove member: %v", err)
	}

	return &emptypb.Empty{}, nil
}
