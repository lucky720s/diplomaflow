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

func (h *Handler) GetProject(ctx context.Context, req *project_pb.GetProjectRequest) (*project_pb.GetProjectResponse, error) {
	project, studentIDs, err := h.repo.GetProjectById(ctx, req.GetProjectId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not get project: %v", err)
	}
	return &project_pb.GetProjectResponse{
		Project: &project_pb.Project{
			Id:           project.ID,
			Topic:        project.Topic,
			SupervisorId: project.SupervisorID,
			DepartmentId: project.DepartmentID,
			StudentIds:   studentIDs,
		},
	}, nil
}

func (h *Handler) ListProjects(ctx context.Context, req *project_pb.ListProjectsRequest) (*project_pb.ListProjectsResponse, error) {
	projects, err := h.repo.ListProjects(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not list projects: %v", err)
	}
	var resProjects []*project_pb.Project
	for _, p := range projects {
		resProjects = append(resProjects, &project_pb.Project{
			Id:           p.ID,
			Topic:        p.Topic,
			SupervisorId: p.SupervisorID,
			DepartmentId: p.DepartmentID,
		})
	}
	return &project_pb.ListProjectsResponse{Projects: resProjects}, nil
}

func (h *Handler) UpdateProject(ctx context.Context, req *project_pb.UpdateProjectRequest) (*project_pb.UpdateProjectResponse, error) {
	project, err := h.repo.UpdateProject(ctx, req.GetProjectId(), req.GetTopic())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not update project: %v", err)
	}
	_, studentIDs, _ := h.repo.GetProjectById(ctx, req.GetProjectId())
	return &project_pb.UpdateProjectResponse{
		Project: &project_pb.Project{
			Id:           project.ID,
			Topic:        project.Topic,
			SupervisorId: project.SupervisorID,
			DepartmentId: project.DepartmentID,
			StudentIds:   studentIDs,
		},
	}, nil
}

func (h *Handler) DeleteProject(ctx context.Context, req *project_pb.DeleteProjectRequest) (*project_pb.DeleteProjectResponse, error) {
	err := h.repo.DeleteProject(ctx, req.GetProjectId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not delete project: %v", err)
	}
	return &project_pb.DeleteProjectResponse{Success: true}, nil
}
