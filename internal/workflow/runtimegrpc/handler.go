package runtimegrpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/lucky720s/diplomaflow/internal/workflow"
	"github.com/lucky720s/diplomaflow/internal/workflow/engine"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	*workflow.Handler

	eng           *engine.WorkflowEngine
	reviewSvc     *workflow.ReviewService
	projectClient projectv1.ProjectServiceClient
	logger        *zap.Logger
}

func New(base *workflow.Handler, eng *engine.WorkflowEngine, reviewSvc *workflow.ReviewService, projectClient projectv1.ProjectServiceClient, logger *zap.Logger) *Handler {
	return &Handler{Handler: base, eng: eng, reviewSvc: reviewSvc, projectClient: projectClient, logger: logger}
}

type callerInfo struct {
	UserID      int64
	UserRole    string
	InternalSvc string
}

func getCaller(ctx context.Context) callerInfo {
	var ci callerInfo
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok || md == nil {
		return ci
	}

	if v := md.Get("x-user-id"); len(v) > 0 {
		if id, err := strconv.ParseInt(v[0], 10, 64); err == nil {
			ci.UserID = id
		}
	}
	if v := md.Get("x-user-role"); len(v) > 0 {
		ci.UserRole = v[0]
	}
	if v := md.Get("x-internal-service"); len(v) > 0 {
		ci.InternalSvc = v[0]
	}
	return ci
}

func withInternalProjectCtx(ctx context.Context, ci callerInfo) context.Context {
	// Всегда помечаем, что это внутренний сервисный вызов,
	// но сохраняем “кто именно” инициировал действие (audit/conditions).
	out := metadata.AppendToOutgoingContext(ctx, "x-internal-service", "workflow_service")
	if ci.UserID > 0 {
		out = metadata.AppendToOutgoingContext(out, "x-user-id", fmt.Sprintf("%d", ci.UserID))
	}
	if ci.UserRole != "" {
		out = metadata.AppendToOutgoingContext(out, "x-user-role", ci.UserRole)
	}
	return out
}

func (h *Handler) GetAvailableTransitions(ctx context.Context, req *workflowv1.GetAvailableTransitionsRequest) (*workflowv1.GetAvailableTransitionsResponse, error) {
	if req.ProjectId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}
	// current_state_id может прийти 0 — тогда возьмём из runtime
	if req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.UserRole == "" {
		// Это поле есть в proto request; gateway обязан его передавать.
		return nil, status.Error(codes.InvalidArgument, "user_role is required")
	}

	ci := getCaller(ctx)

	// Security: если это не internal вызов, запрещаем подмену user_id
	if ci.InternalSvc == "" {
		if ci.UserID == 0 || ci.UserRole == "" {
			return nil, status.Error(codes.Unauthenticated, "missing auth metadata")
		}
		if ci.UserID != req.UserId {
			return nil, status.Error(codes.PermissionDenied, "user_id mismatch")
		}
		// user_role в request должен совпадать с ролью токена (иначе тоже подмена)
		if ci.UserRole != req.UserRole {
			return nil, status.Error(codes.PermissionDenied, "user_role mismatch")
		}
	}

	snap, err := h.projectClient.GetProjectRuntime(withInternalProjectCtx(ctx, ci), &projectv1.GetProjectRuntimeRequest{
		ProjectId: req.ProjectId,
	})
	if err != nil {
		return nil, err
	}

	projectData := map[string]interface{}{}
	if snap.Data != nil {
		projectData = snap.Data.AsMap()
	}

	currentStateID := req.CurrentStateId
	if currentStateID == 0 {
		currentStateID = snap.CurrentStateId
	}

	available, err := h.eng.GetAvailableTransitions(ctx, req.ProjectId, currentStateID, req.UserId, req.UserRole, projectData)
	if err != nil {
		return nil, err
	}

	resp := &workflowv1.GetAvailableTransitionsResponse{}
	for _, at := range available {
		resp.Transitions = append(resp.Transitions, &workflowv1.AvailableTransition{
			Transition: &workflowv1.Transition{
				Id:          at.Transition.ID,
				WorkflowId:  at.Transition.WorkflowID,
				EventName:   at.Transition.EventName,
				DisplayName: at.Transition.DisplayName,
				FromStateId: at.Transition.FromStateID,
				ToStateId:   at.Transition.ToStateID,
				ButtonLabel: at.Transition.ButtonLabel,
				ButtonColor: at.Transition.ButtonColor,
				ConfirmText: at.Transition.ConfirmText,
			},
			CanExecute:          at.CanExecute,
			BlockedReason:       at.BlockedReason,
			MissingRequirements: at.MissingRequirements,
		})
	}

	return resp, nil
}

func (h *Handler) ExecuteTransition(ctx context.Context, req *workflowv1.ExecuteTransitionRequest) (*workflowv1.ExecuteTransitionResponse, error) {
	if req.ProjectId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}
	if req.TransitionId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "transition_id is required")
	}
	if req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	ci := getCaller(ctx)

	// Security: unless internal, require auth metadata and match with request user_id
	if ci.InternalSvc == "" {
		if ci.UserID == 0 || ci.UserRole == "" {
			return nil, status.Error(codes.Unauthenticated, "missing auth metadata (x-user-id/x-user-role)")
		}
		if ci.UserID != req.UserId {
			return nil, status.Error(codes.PermissionDenied, "user_id mismatch")
		}
	}

	// We require role to evaluate conditions consistently.
	// If internal call didn't provide role, it's an integration bug.
	if ci.UserRole == "" {
		return nil, status.Error(codes.InvalidArgument, "x-user-role metadata is required")
	}

	// Fetch project runtime snapshot (internal call)
	snap, err := h.projectClient.GetProjectRuntime(withInternalProjectCtx(ctx, ci), &projectv1.GetProjectRuntimeRequest{
		ProjectId: req.ProjectId,
	})
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{}
	if req.Payload != nil {
		payload = req.Payload.AsMap()
	}

	projectData := map[string]interface{}{}
	if snap.Data != nil {
		projectData = snap.Data.AsMap()
	}

	// Execute in engine
	res, err := h.eng.ExecuteTransition(ctx, &engine.ExecuteTransitionRequest{
		ProjectID:      req.ProjectId,
		TransitionID:   req.TransitionId,
		CurrentStateID: snap.CurrentStateId,
		UserID:         req.UserId,
		UserRole:       ci.UserRole,
		DepartmentID:   snap.DepartmentId,
		UniversityID:   snap.UniversityId,
		TeamID:         snap.TeamId,
		ProjectData:    projectData,
		Payload:        payload,
	})
	if err != nil {
		// Map domain errors to proper gRPC codes
		switch {
		case errors.Is(err, engine.ErrTransitionNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		case errors.Is(err, engine.ErrTransitionNotAllowed):
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		case errors.Is(err, engine.ErrConditionFailed):
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	// Convert patch
	var patchStruct *structpb.Struct
	if res.DataPatch != nil {
		b, _ := json.Marshal(res.DataPatch)
		m := map[string]interface{}{}
		_ = json.Unmarshal(b, &m)
		patchStruct, _ = structpb.NewStruct(m)
	}

	var deadline *timestamppb.Timestamp
	if res.NewDeadline != nil {
		deadline = timestamppb.New(*res.NewDeadline)
	}

	var postGroups []*projectv1.PostCommitActionGroup
	for _, g := range res.PostActions {
		if g.Trigger == "" || len(g.ActionIDs) == 0 {
			continue
		}
		postGroups = append(postGroups, &projectv1.PostCommitActionGroup{
			Trigger:   g.Trigger,
			ActionIds: g.ActionIDs,
		})
	}

	// Commit transition back to project_service (internal)
	_, err = h.projectClient.CommitTransition(withInternalProjectCtx(ctx, ci), &projectv1.CommitTransitionRequest{
		ProjectId:           req.ProjectId,
		ExpectedFromStateId: snap.CurrentStateId,
		TransitionId:        req.TransitionId,
		EventName:           res.EventName,
		ToStateId:           res.NewStateID,
		ToStateName:         res.NewStateName,
		ChangedBy:           req.UserId,
		Comment:             fmt.Sprintf("event=%s", res.EventName),
		NewDeadlineAt:       deadline,
		DataPatch:           patchStruct,
		PostActions:         postGroups,
		SetStatus:           res.SetStatus,
	})
	if err != nil {
		h.logger.Error("CommitTransition failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "commit failed: %v", err)
	}

	return &workflowv1.ExecuteTransitionResponse{
		Success:         true,
		NewStateId:      res.NewStateID,
		NewStateName:    res.NewStateName,
		ExecutedActions: res.ExecutedActionNames,
	}, nil
}
func (h *Handler) SubmitReview(ctx context.Context, req *workflowv1.SubmitReviewRequest) (*workflowv1.SubmitReviewResponse, error) {
	if req.ProjectId <= 0 || req.StateId <= 0 || req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id, state_id, user_id are required")
	}
	if req.RoleSlug == "" {
		return nil, status.Error(codes.InvalidArgument, "role_slug is required")
	}

	ci := getCaller(ctx)
	if ci.InternalSvc == "" {
		if ci.UserID == 0 || ci.UserRole == "" {
			return nil, status.Error(codes.Unauthenticated, "missing auth metadata")
		}
		if ci.UserID != req.UserId {
			return nil, status.Error(codes.PermissionDenied, "user_id mismatch")
		}
	}

	sreq := &workflow.SubmitReviewRequest{
		ProjectID:  req.ProjectId,
		StateID:    req.StateId,
		ReviewerID: req.UserId,
		RoleSlug:   req.RoleSlug,
		Decision:   req.Decision,
		Comment:    req.Comment,
	}
	if req.HasScore {
		s := req.Score
		sreq.Score = &s
	}

	resp, err := h.reviewSvc.SubmitReview(ctx, sreq)
	if err != nil {
		switch {
		case errors.Is(err, workflow.ErrReviewNotAllowed):
			return nil, status.Error(codes.PermissionDenied, err.Error())
		case errors.Is(err, workflow.ErrInvalidScore), errors.Is(err, workflow.ErrInvalidDecision):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &workflowv1.SubmitReviewResponse{
		AllVoted:        resp.AllVoted,
		ReadyToFinalize: resp.ReadyToFinalize,
		Summary:         summaryToProto(resp.Summary),
	}, nil
}

func (h *Handler) GetStateReviews(ctx context.Context, req *workflowv1.GetStateReviewsRequest) (*workflowv1.GetStateReviewsResponse, error) {
	if req.ProjectId <= 0 || req.StateId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id and state_id are required")
	}

	reviews, err := h.reviewSvc.GetReviews(ctx, req.ProjectId, req.StateId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	summary, err := h.reviewSvc.GetSummary(ctx, req.ProjectId, req.StateId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var protoReviews []*workflowv1.StateReviewProto
	for _, r := range reviews {
		pr := &workflowv1.StateReviewProto{
			Id:         r.ID,
			ProjectId:  r.ProjectID,
			StateId:    r.StateID,
			ReviewerId: r.ReviewerID,
			RoleSlug:   r.RoleSlug,
			Decision:   r.Decision,
			Comment:    r.Comment,
			CreatedAt:  timestamppb.New(r.CreatedAt),
			UpdatedAt:  timestamppb.New(r.UpdatedAt),
		}
		if r.Score != nil {
			pr.Score = *r.Score
			pr.HasScore = true
		}
		protoReviews = append(protoReviews, pr)
	}

	return &workflowv1.GetStateReviewsResponse{
		Reviews: protoReviews,
		Summary: summaryToProto(*summary),
	}, nil
}

func summaryToProto(s workflow.ReviewSummary) *workflowv1.ReviewSummaryProto {
	p := &workflowv1.ReviewSummaryProto{
		TotalReviews:  int32(s.TotalReviews),
		AverageScore:  s.AverageScore,
		ApprovedCount: int32(s.ApprovedCount),
		RejectedCount: int32(s.RejectedCount),
		AllVoted:      s.AllVoted,
		Result:        s.Result,
	}
	if s.FinalScore != nil {
		p.FinalScore = *s.FinalScore
		p.HasFinalScore = true
	}
	return p
}

func (h *Handler) CanExecuteTransition(ctx context.Context, req *workflowv1.CanExecuteTransitionRequest) (*workflowv1.CanExecuteTransitionResponse, error) {
	if req.ProjectId <= 0 || req.TransitionId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "project_id and transition_id are required")
	}

	ci := getCaller(ctx)

	snap, err := h.projectClient.GetProjectRuntime(
		withInternalProjectCtx(ctx, ci),
		&projectv1.GetProjectRuntimeRequest{ProjectId: req.ProjectId},
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get project runtime: %v", err)
	}

	projectData := map[string]interface{}{}
	if snap.Data != nil {
		projectData = snap.Data.AsMap()
	}

	userID := ci.UserID
	userRole := ci.UserRole
	if ci.InternalSvc != "" {
		if req.UserId > 0 {
			userID = req.UserId
		}
		if req.UserRole != "" {
			userRole = req.UserRole
		}
	}

	currentStateID := req.CurrentStateId
	if currentStateID == 0 {
		currentStateID = snap.CurrentStateId
	}

	available, err := h.eng.GetAvailableTransitions(ctx, req.ProjectId, currentStateID, userID, userRole, projectData)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get transitions: %v", err)
	}

	for _, at := range available {
		if at.Transition != nil && at.Transition.ID == req.TransitionId {
			return &workflowv1.CanExecuteTransitionResponse{
				CanExecute:          at.CanExecute,
				BlockedReason:       at.BlockedReason,
				MissingRequirements: at.MissingRequirements,
			}, nil
		}
	}

	return &workflowv1.CanExecuteTransitionResponse{
		CanExecute:    false,
		BlockedReason: "transition not available from current state",
	}, nil
}
