package project

import (
	"context"

	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	projectv1.UnimplementedProjectServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateProject(ctx context.Context, req *projectv1.CreateProjectRequest) (*projectv1.CreateProjectResponse, error) {
	if req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	if req.StudentId == 0 {
		return nil, status.Error(codes.InvalidArgument, "student_id is required")
	}
	if req.WorkflowName == "" {
		return nil, status.Error(codes.InvalidArgument, "workflow_name is required")
	}

	project, err := h.service.CreateProject(ctx, req.Title, req.Description, req.StudentId, req.WorkflowName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create project: %v", err)
	}

	return &projectv1.CreateProjectResponse{
		ProjectId: int64(project.ID),
		Status:    project.Status,
	}, nil
}

func (h *Handler) GetProject(ctx context.Context, req *projectv1.GetProjectRequest) (*projectv1.GetProjectResponse, error) {
	if req.ProjectId == 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}

	project, err := h.service.GetProject(ctx, uint64(req.ProjectId))
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "project not found: %v", err)
	}

	var historyPb []*projectv1.StateHistory
	for _, h := range project.History {
		historyPb = append(historyPb, &projectv1.StateHistory{
			StateName: h.StateName,
			Status:    h.Status,
			ChangedBy: h.ChangedBy,
			Comment:   h.Comment,
			Timestamp: h.CreatedAt.String(),
		})
	}

	return &projectv1.GetProjectResponse{
		ProjectId:    int64(project.ID),
		Title:        project.Title,
		Description:  project.Description,
		StudentId:    project.StudentID,
		TeamId:       project.TeamID,
		WorkflowName: project.WorkflowName,
		CurrentState: project.CurrentState,
		Status:       project.Status,
		History:      historyPb,
	}, nil
}

func (h *Handler) GetStudentProjects(ctx context.Context, req *projectv1.GetStudentProjectsRequest) (*projectv1.GetStudentProjectsResponse, error) {
	if req.StudentId == 0 {
		return nil, status.Error(codes.InvalidArgument, "student_id is required")
	}

	projects, err := h.service.GetStudentProjects(ctx, req.StudentId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list projects: %v", err)
	}

	var responseProjects []*projectv1.ProjectPreview
	for _, p := range projects {
		responseProjects = append(responseProjects, &projectv1.ProjectPreview{
			ProjectId:    int64(p.ID),
			Title:        p.Title,
			Status:       p.Status,
			CurrentState: p.CurrentState,
		})
	}

	return &projectv1.GetStudentProjectsResponse{
		Projects: responseProjects,
	}, nil
}
