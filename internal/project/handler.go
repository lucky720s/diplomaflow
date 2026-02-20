package project

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ProjectUseCase interface {
	CreateProject(ctx context.Context, req *projectv1.CreateProjectRequest) (*projectv1.CreateProjectResponse, error)
	GetProject(ctx context.Context, id int64) (*Project, error)
	GetStudentProjects(ctx context.Context, studentID int64) ([]*Project, error)

	PerformAction(ctx context.Context, projectID int64, actionName string, payload map[string]interface{}, userID int64, userRole string) (*Project, error)

	GetProjectRuntime(ctx context.Context, projectID int64) (*projectv1.GetProjectRuntimeResponse, error)
	CommitTransition(ctx context.Context, req *projectv1.CommitTransitionRequest) (*projectv1.CommitTransitionResponse, error)
}

type Handler struct {
	projectv1.UnimplementedProjectServiceServer
	service ProjectUseCase
}

func NewHandler(service ProjectUseCase) *Handler { return &Handler{service: service} }

// ---- metadata helpers ----

func getInternalService(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if v := md.Get("x-internal-service"); len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

func getRequesterID(ctx context.Context) int64 {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if v := md.Get("x-user-id"); len(v) > 0 {
			id, _ := strconv.ParseInt(v[0], 10, 64)
			return id
		}
	}
	return 0
}

func getRequesterRole(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if v := md.Get("x-user-role"); len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

func requireInternal(ctx context.Context, allow ...string) error {
	svc := getInternalService(ctx)
	for _, a := range allow {
		if svc == a {
			return nil
		}
	}
	return status.Error(codes.PermissionDenied, "forbidden")
}

// ---- RPCs ----

func (h *Handler) CreateProject(ctx context.Context, req *projectv1.CreateProjectRequest) (*projectv1.CreateProjectResponse, error) {
	// project should be created only after supervisor approve/claim (admin flow)
	if err := requireInternal(ctx, "admin_service"); err != nil {
		return nil, err
	}
	if req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	if req.StudentId <= 0 || req.UniversityId <= 0 || req.DepartmentId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "student_id, university_id, department_id are required")
	}
	return h.service.CreateProject(ctx, req)
}

func (h *Handler) GetProject(ctx context.Context, req *projectv1.GetProjectRequest) (*projectv1.GetProjectResponse, error) {
	if req.ProjectId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}

	var p *Project
	var err error

	internalSvc := getInternalService(ctx)

	if internalSvc != "" {
		if permErr := requireInternal(ctx, "admin_service", "workflow_service"); permErr != nil {
			return nil, permErr
		}
		p, err = h.service.GetProject(ctx, req.ProjectId)
		if err != nil {
			// если у тебя service/repo умеет различать not found — мапь тут
			return nil, status.Errorf(codes.NotFound, "project not found: %v", err)
		}
		if p == nil {
			return nil, status.Error(codes.NotFound, "project not found")
		}
	} else {
		uid := getRequesterID(ctx)
		if uid == 0 {
			return nil, status.Error(codes.Unauthenticated, "unauthorized")
		}

		visible, err2 := h.service.GetStudentProjects(ctx, uid)
		if err2 != nil {
			return nil, status.Errorf(codes.Internal, "failed: %v", err2)
		}

		for _, pr := range visible {
			if pr != nil && pr.ID == req.ProjectId {
				p = pr
				break
			}
		}
		if p == nil {
			return nil, status.Error(codes.NotFound, "project not found")
		}
	}

	var hist []*projectv1.StateHistory
	for _, hh := range p.History {
		changedBy := int64(0)
		if hh.ChangedBy != nil {
			changedBy = *hh.ChangedBy
		}
		hist = append(hist, &projectv1.StateHistory{
			StateName: hh.ToStateName,
			Status:    hh.Status,
			ChangedBy: changedBy,
			Comment:   hh.Comment,
			Timestamp: hh.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	return &projectv1.GetProjectResponse{
		ProjectId:    p.ID,
		Title:        p.Title,
		Description:  p.Description,
		StudentId:    p.StudentID,
		TeamId:       p.TeamID,
		WorkflowName: p.WorkflowName,
		CurrentState: p.CurrentStateName,
		Status:       p.Status,
		History:      hist,
	}, nil
}

func (h *Handler) GetStudentProjects(ctx context.Context, req *projectv1.GetStudentProjectsRequest) (*projectv1.GetStudentProjectsResponse, error) {
	if req.StudentId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "student_id is required")
	}

	internalSvc := getInternalService(ctx)
	if internalSvc != "" {
		if err := requireInternal(ctx, "admin_service", "workflow_service"); err != nil {
			return nil, err
		}
	} else {
		uid := getRequesterID(ctx)
		if uid == 0 {
			return nil, status.Error(codes.Unauthenticated, "unauthorized")
		}
		if uid != req.StudentId {
			return nil, status.Error(codes.PermissionDenied, "forbidden")
		}
	}

	list, err := h.service.GetStudentProjects(ctx, req.StudentId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed: %v", err)
	}

	resp := &projectv1.GetStudentProjectsResponse{}
	for _, p := range list {
		pp := &projectv1.ProjectPreview{
			ProjectId:        p.ID,
			Title:            p.Title,
			Status:           p.Status,
			CurrentState:     p.CurrentStateName,
			TeamId:           p.TeamID,
			WorkflowId:       p.WorkflowID,
			WorkflowName:     p.WorkflowName,
			CurrentStateId:   p.CurrentStateID,
			CurrentStateName: p.CurrentStateName,
		}
		if p.DeadlineAt != nil {
			pp.DeadlineAt = timestamppb.New(*p.DeadlineAt)
		}
		resp.Projects = append(resp.Projects, pp)
	}
	return resp, nil
}

func (h *Handler) PerformAction(ctx context.Context, req *projectv1.PerformActionRequest) (*projectv1.PerformActionResponse, error) {
	if req.ProjectId <= 0 || req.ActionName == "" {
		return nil, status.Error(codes.InvalidArgument, "project_id and action_name are required")
	}

	payload := map[string]interface{}{}
	if req.Payload != nil {
		b, _ := req.Payload.MarshalJSON()
		_ = json.Unmarshal(b, &payload)
	}

	userID := getRequesterID(ctx)
	role := getRequesterRole(ctx)

	if userID == 0 {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	p, err := h.service.PerformAction(ctx, req.ProjectId, req.ActionName, payload, userID, role)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "perform action failed: %v", err)
	}
	return &projectv1.PerformActionResponse{ProjectId: p.ID, NewState: p.CurrentStateName}, nil
}

func (h *Handler) GetProjectRuntime(ctx context.Context, req *projectv1.GetProjectRuntimeRequest) (*projectv1.GetProjectRuntimeResponse, error) {
	if err := requireInternal(ctx, "workflow_service", "admin_service", "api_gateway"); err != nil {
		return nil, err
	}

	if req.ProjectId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}
	return h.service.GetProjectRuntime(ctx, req.ProjectId)
}

func (h *Handler) CommitTransition(ctx context.Context, req *projectv1.CommitTransitionRequest) (*projectv1.CommitTransitionResponse, error) {
	// runtime API: internal-only (workflow)
	if err := requireInternal(ctx, "workflow_service"); err != nil {
		return nil, err
	}
	if req.ProjectId <= 0 || req.ExpectedFromStateId <= 0 || req.ToStateId <= 0 || req.EventName == "" {
		return nil, status.Error(codes.InvalidArgument, "project_id, expected_from_state_id, to_state_id, event_name are required")
	}
	return h.service.CommitTransition(ctx, req)
}
