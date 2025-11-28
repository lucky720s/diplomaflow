package project

import (
	"context"
	"log"

	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Handler struct {
	projectv1.UnimplementedProjectServiceServer
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) CreateProject(ctx context.Context, req *projectv1.CreateProjectRequest) (*projectv1.Project, error) {
	newProject, err := h.repo.CreateProject(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create project: %v", err)
	}
	return toProtoProject(newProject), nil
}

func (h *Handler) PerformStateAction(ctx context.Context, req *projectv1.PerformStateActionRequest) (*emptypb.Empty, error) {
	err := h.repo.PerformStateAction(ctx, req)
	if err != nil {
		log.Printf("Error in PerformStateAction: %v", err)
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) GetProject(ctx context.Context, req *projectv1.GetProjectRequest) (*projectv1.Project, error) {
	p, err := h.repo.GetProject(ctx, req.GetProjectId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "project not found")
	}
	return toProtoProject(p), nil
}

func toProtoProject(p *Project) *projectv1.Project {
	return &projectv1.Project{
		Id:             p.ID,
		Title:          p.Title,
		WorkflowId:     p.WorkflowID,
		CurrentStateId: p.CurrentStateID,
		Status:         p.Status,
	}
}
func (h *Handler) ListProjects(ctx context.Context, req *projectv1.ListProjectsRequest) (*projectv1.ListProjectsResponse, error) {
	projects, err := h.repo.ListProjects(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list projects: %v", err)
	}

	protoProjects := make([]*projectv1.Project, len(projects))
	for i, p := range projects {
		protoProjects[i] = toProtoProject(p)
	}

	return &projectv1.ListProjectsResponse{Projects: protoProjects}, nil
}
