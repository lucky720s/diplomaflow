package runtimegrpc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lucky720s/diplomaflow/internal/workflow"
	"github.com/lucky720s/diplomaflow/internal/workflow/engine"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	// embed base CRUD handler (no engine import inside workflow package)
	*workflow.Handler

	eng           *engine.WorkflowEngine
	projectClient projectv1.ProjectServiceClient
	logger        *zap.Logger
}

func New(service *workflow.Service, eng *engine.WorkflowEngine, projectClient projectv1.ProjectServiceClient, logger *zap.Logger) *Handler {
	return &Handler{
		Handler:       workflow.NewHandler(service, logger),
		eng:           eng,
		projectClient: projectClient,
		logger:        logger,
	}
}

func (h *Handler) GetAvailableTransitions(ctx context.Context, req *workflowv1.GetAvailableTransitionsRequest) (*workflowv1.GetAvailableTransitionsResponse, error) {
	// runtime snapshot always from project_service (source of truth)
	snap, err := h.projectClient.GetProjectRuntime(ctx, &projectv1.GetProjectRuntimeRequest{ProjectId: req.ProjectId})
	if err != nil {
		return nil, err
	}

	projectData := map[string]interface{}{}
	if snap.Data != nil {
		projectData = snap.Data.AsMap()
	}

	available, err := h.eng.GetAvailableTransitions(ctx, req.ProjectId, req.CurrentStateId, req.UserId, req.UserRole, projectData)
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
	// get role from metadata (project_service must forward it)
	userRole := ""
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if v := md.Get("x-user-role"); len(v) > 0 {
			userRole = v[0]
		}
	}

	snap, err := h.projectClient.GetProjectRuntime(ctx, &projectv1.GetProjectRuntimeRequest{ProjectId: req.ProjectId})
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{}
	if req.Payload != nil {
		payload = req.Payload.AsMap()
	}

	res, err := h.eng.ExecuteTransition(ctx, &engine.ExecuteTransitionRequest{
		ProjectID:      req.ProjectId,
		TransitionID:   req.TransitionId,
		CurrentStateID: snap.CurrentStateId,
		UserID:         req.UserId,
		UserRole:       userRole,
		DepartmentID:   snap.DepartmentId,
		ProjectData:    snap.Data.AsMap(),
		Payload:        payload,
	})
	if err != nil {
		return &workflowv1.ExecuteTransitionResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// DataPatch -> Struct
	patchStruct := (*structpb.Struct)(nil)
	if res.DataPatch != nil {
		// normalize via JSON to avoid non-struct values
		b, _ := json.Marshal(res.DataPatch)
		m := map[string]interface{}{}
		_ = json.Unmarshal(b, &m)
		patchStruct, _ = structpb.NewStruct(m)
	}

	// Post actions -> proto groups
	var postGroups []*projectv1.PostCommitActionGroup
	for _, g := range res.PostCommitActions {
		postGroups = append(postGroups, &projectv1.PostCommitActionGroup{
			Trigger:   g.Trigger,
			ActionIds: g.ActionIDs,
		})
	}

	commitReq := &projectv1.CommitTransitionRequest{
		ProjectId:           req.ProjectId,
		ExpectedFromStateId: snap.CurrentStateId,
		TransitionId:        req.TransitionId,
		EventName:           res.EventName,
		ToStateId:           res.NewStateID,
		ToStateName:         res.NewStateName,
		ChangedBy:           req.UserId,
		Comment:             fmt.Sprintf("event=%s", res.EventName),
		DataPatch:           patchStruct,
		PostActions:         postGroups,
		SetStatus:           res.SetStatus,
	}

	if res.NewDeadline != nil {
		commitReq.NewDeadlineAt = timestamppb.New(*res.NewDeadline)
	}

	_, err = h.projectClient.CommitTransition(ctx, commitReq)
	if err != nil {
		return &workflowv1.ExecuteTransitionResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("commit failed: %v", err),
		}, nil
	}

	return &workflowv1.ExecuteTransitionResponse{
		Success:         true,
		NewStateId:      res.NewStateID,
		NewStateName:    res.NewStateName,
		ExecutedActions: res.ExecutedActionNames,
	}, nil
}
