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
	if req.ProjectId == 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}

	teamID, err := h.service.CreateTeam(ctx, req.Name, req.ProjectId, req.MemberIds)
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

	return &pb.GetTeamResponse{
		TeamId:    int64(team.ID),
		Name:      team.Name,
		ProjectId: team.ProjectID,
		Members:   pbMembers,
	}, nil
}

func (h *Handler) GetAvailableStudents(ctx context.Context, req *pb.GetAvailableStudentsRequest) (*pb.GetAvailableStudentsResponse, error) {
	if req.UniversityId == 0 {
		return nil, status.Error(codes.InvalidArgument, "university_id is required")
	}

	students, err := h.service.GetAvailableStudents(ctx, req.UniversityId)
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
