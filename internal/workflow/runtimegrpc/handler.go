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
	*workflow.Handler

	eng           *engine.WorkflowEngine
	projectClient projectv1.ProjectServiceClient
	logger        *zap.Logger
}

func New(base *workflow.Handler, eng *engine.WorkflowEngine, projectClient projectv1.ProjectServiceClient, logger *zap.Logger) *Handler {
	return &Handler{Handler: base, eng: eng, projectClient: projectClient, logger: logger}
}

func (h *Handler) GetAvailableTransitions(ctx context.Context, req *workflowv1.GetAvailableTransitionsRequest) (*workflowv1.GetAvailableTransitionsResponse, error) {
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

	projectData := map[string]interface{}{}
	if snap.Data != nil {
		projectData = snap.Data.AsMap()
	}

	res, err := h.eng.ExecuteTransition(ctx, &engine.ExecuteTransitionRequest{
		ProjectID:      req.ProjectId,
		TransitionID:   req.TransitionId,
		CurrentStateID: snap.CurrentStateId,
		UserID:         req.UserId,
		UserRole:       userRole,
		DepartmentID:   snap.DepartmentId,
		UniversityID:   snap.UniversityId,
		TeamID:         snap.TeamId,
		ProjectData:    projectData,
		Payload:        payload,
	})
	if err != nil {
		return &workflowv1.ExecuteTransitionResponse{Success: false, ErrorMessage: err.Error()}, nil
	}

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

	_, err = h.projectClient.CommitTransition(ctx, &projectv1.CommitTransitionRequest{
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
		return &workflowv1.ExecuteTransitionResponse{Success: false, ErrorMessage: fmt.Sprintf("commit failed: %v", err)}, nil
	}

	return &workflowv1.ExecuteTransitionResponse{
		Success:         true,
		NewStateId:      res.NewStateID,
		NewStateName:    res.NewStateName,
		ExecutedActions: res.ExecutedActionNames,
	}, nil
}
