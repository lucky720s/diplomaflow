package project

import (
	"context"
	"fmt"

	auth_pb "github.com/lucky720s/diplomaflow/pkg/protobuf/auth"
	project_pb "github.com/lucky720s/diplomaflow/pkg/protobuf/project"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	project, err := h.repo.CreateProject(ctx, req.GetTopic(), req.GetSupervisorId(), req.GetStudentIds(), req.GetDepartmentId())
	if err != nil {
		return nil, err
	}
	return &project_pb.CreateProjectResponse{ProjectId: project.ID}, nil
}
