package runtimegrpc

import (
	"context"
	"errors"
	"strconv"

	"github.com/lucky720s/diplomaflow/internal/workflow"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AdminRuntimeHandler extends the runtime gRPC handler with admin project workflow RPCs.
type AdminRuntimeHandler struct {
	*Handler
	adminSvc *workflow.AdminRuntimeService
	logger   *zap.Logger
}

func NewAdminRuntimeHandler(base *Handler, adminSvc *workflow.AdminRuntimeService, logger *zap.Logger) *AdminRuntimeHandler {
	return &AdminRuntimeHandler{Handler: base, adminSvc: adminSvc, logger: logger}
}

// requireAdmin checks that the caller has admin or superadmin role from metadata.
func requireAdmin(ctx context.Context) (int64, string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	var userID int64
	var role string

	// Internal services are allowed unconditionally (they set x-internal-service)
	if v := md.Get("x-internal-service"); len(v) > 0 {
		// still read actor identity
		if v2 := md.Get("x-user-id"); len(v2) > 0 {
			userID, _ = strconv.ParseInt(v2[0], 10, 64)
		}
		if v2 := md.Get("x-user-role"); len(v2) > 0 {
			role = v2[0]
		}
		return userID, role, nil
	}

	if v := md.Get("x-user-id"); len(v) > 0 {
		userID, _ = strconv.ParseInt(v[0], 10, 64)
	}
	if v := md.Get("x-user-role"); len(v) > 0 {
		role = v[0]
	}

	if userID == 0 {
		return 0, "", status.Error(codes.Unauthenticated, "unauthorized: missing x-user-id")
	}
	if role != "admin" && role != "superadmin" {
		return 0, "", status.Errorf(codes.PermissionDenied, "admin role required, got %q", role)
	}
	return userID, role, nil
}

func (h *AdminRuntimeHandler) GetProjectWorkflowState(ctx context.Context, req *workflowv1.GetProjectWorkflowStateRequest) (*workflowv1.ProjectWorkflowStateResponse, error) {
	if req.ProjectId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}
	// Read access: admin or internal service
	if _, _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	resp, err := h.adminSvc.GetProjectWorkflowState(ctx, req.ProjectId)
	if err != nil {
		return nil, mapAdminErr(err)
	}
	return resp, nil
}

func (h *AdminRuntimeHandler) AdvanceProjectWorkflow(ctx context.Context, req *workflowv1.AdvanceProjectWorkflowRequest) (*workflowv1.ProjectWorkflowStateResponse, error) {
	if req.ProjectId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}
	actorID, actorRole, err := requireAdmin(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := h.adminSvc.AdvanceProjectWorkflow(ctx, req, actorID, actorRole)
	if err != nil {
		return nil, mapAdminErr(err)
	}
	return resp, nil
}

func (h *AdminRuntimeHandler) RollbackProjectWorkflow(ctx context.Context, req *workflowv1.RollbackProjectWorkflowRequest) (*workflowv1.ProjectWorkflowStateResponse, error) {
	if req.ProjectId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}
	actorID, actorRole, err := requireAdmin(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := h.adminSvc.RollbackProjectWorkflow(ctx, req, actorID, actorRole)
	if err != nil {
		return nil, mapAdminErr(err)
	}
	return resp, nil
}

func (h *AdminRuntimeHandler) SetProjectWorkflowState(ctx context.Context, req *workflowv1.SetProjectWorkflowStateRequest) (*workflowv1.ProjectWorkflowStateResponse, error) {
	if req.ProjectId <= 0 || req.ToStateId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id and to_state_id are required")
	}
	actorID, actorRole, err := requireAdmin(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := h.adminSvc.SetProjectWorkflowState(ctx, req, actorID, actorRole)
	if err != nil {
		return nil, mapAdminErr(err)
	}
	return resp, nil
}

func (h *AdminRuntimeHandler) ListProjectWorkflowHistory(ctx context.Context, req *workflowv1.ListProjectWorkflowHistoryRequest) (*workflowv1.ListProjectWorkflowHistoryResponse, error) {
	if req.ProjectId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}
	if _, _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	resp, err := h.adminSvc.ListProjectWorkflowHistory(ctx, req)
	if err != nil {
		return nil, mapAdminErr(err)
	}
	return resp, nil
}

func (h *AdminRuntimeHandler) ListWorkflowProjects(ctx context.Context, req *workflowv1.ListWorkflowProjectsRequest) (*workflowv1.ListWorkflowProjectsResponse, error) {
	if _, _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}

	resp, err := h.adminSvc.ListWorkflowProjects(ctx, req)
	if err != nil {
		return nil, mapAdminErr(err)
	}
	return resp, nil
}

func (h *AdminRuntimeHandler) GetWorkflowDashboardStats(ctx context.Context, req *workflowv1.GetWorkflowDashboardStatsRequest) (*workflowv1.GetWorkflowDashboardStatsResponse, error) {
	if _, _, err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	if req.DepartmentId == 0 && req.WorkflowId == 0 {
		return nil, status.Error(codes.InvalidArgument, "department_id or workflow_id is required")
	}

	resp, err := h.adminSvc.GetWorkflowDashboardStats(ctx, req)
	if err != nil {
		return nil, mapAdminErr(err)
	}
	return resp, nil
}

func mapAdminErr(err error) error {
	if err == nil {
		return nil
	}
	// Already a gRPC status error — pass through
	if _, ok := status.FromError(err); ok {
		return err
	}
	switch {
	case errors.Is(err, workflow.ErrProjectNotFound):
		return status.Errorf(codes.NotFound, "%v", err)
	case errors.Is(err, workflow.ErrInvalidTransition):
		return status.Errorf(codes.FailedPrecondition, "%v", err)
	case errors.Is(err, workflow.ErrRollbackNotAllowed):
		return status.Errorf(codes.FailedPrecondition, "%v", err)
	case errors.Is(err, workflow.ErrForbidden):
		return status.Errorf(codes.PermissionDenied, "%v", err)
	case errors.Is(err, workflow.ErrActiveWFNotConfigured):
		return status.Errorf(codes.FailedPrecondition, "%v", err)
	default:
		return status.Errorf(codes.Internal, "%v", err)
	}
}
