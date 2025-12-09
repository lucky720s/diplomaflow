package team

import (
	"context"

	pb "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	pb.UnimplementedTeamServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateTeam(ctx context.Context, req *pb.CreateTeamRequest) (*pb.CreateTeamResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "team name is required")
	}
	var projectID *int64
	if req.ProjectId != 0 {
		p := req.ProjectId
		projectID = &p
	}
	teamID, err := h.service.CreateTeam(ctx, req.Name, projectID, req.MemberIds, req.LeaderId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create team: %v", err)
	}

	return &pb.CreateTeamResponse{
		TeamId: teamID,
	}, nil
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
			Id:       s.Id,
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
