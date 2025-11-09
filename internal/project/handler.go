package project

import (
	"context"
	"fmt"

	auth_pb "github.com/lucky720s/diplomaflow/pkg/protobuf/auth"
	project_pb "github.com/lucky720s/diplomaflow/pkg/protobuf/project"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Handler struct {
	project_pb.UnimplementedProjectServiceServer
	repo       *ProjectRepository
	authClient auth_pb.AuthServiceClient
}

func NewHandler() *Handler {
	conn, err := grpc.Dial("auth_service:8082", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Sprintf("failed to connect to auth service: %v", err))
	}
	authClient := auth_pb.NewAuthServiceClient(conn)
	return &Handler{repo: NewProjectRepository(), authClient: authClient}
}

func (h *Handler) CreateProject(ctx context.Context, req *project_pb.CreateProjectRequest) (*project_pb.CreateProjectResponse, error) {
	userIDsToCheck := append(req.GetStudentIds(), req.GetSupervisorId())
	for _, id := range userIDsToCheck {
		_, err := h.authClient.GetUser(ctx, &auth_pb.GetUserRequest{UserId: id})
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
		}
	}
	project, err := h.repo.CreateProject(ctx, req.GetTopic(), req.GetSupervisorId(), req.GetStudentIds(), req.GetDepartmentId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not create project: %v", err)
	}
	return &project_pb.CreateProjectResponse{ProjectId: project.ID}, nil
}
