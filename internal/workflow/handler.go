// internal/workflow/handler.go

package workflow

import (
	"context"
	"encoding/json"

	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	workflowv1.UnimplementedWorkflowServiceServer
	service *Service
	logger  *zap.Logger
}

func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// ==================== WORKFLOW CRUD ====================

func (h *Handler) CreateWorkflow(ctx context.Context, req *workflowv1.CreateWorkflowRequest) (*workflowv1.Workflow, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.DepartmentId == 0 {
		return nil, status.Error(codes.InvalidArgument, "department_id is required")
	}

	var settings map[string]interface{}
	if req.Settings != nil {
		settings = req.Settings.AsMap()
	}

	wf, err := h.service.CreateWorkflow(ctx, &CreateWorkflowInput{
		Name:         req.Name,
		Description:  req.Description,
		DepartmentID: req.DepartmentId,
		Settings:     settings,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create workflow: %v", err)
	}

	return h.workflowToProto(wf), nil
}

func (h *Handler) GetWorkflow(ctx context.Context, req *workflowv1.GetWorkflowRequest) (*workflowv1.Workflow, error) {
	var wf *Workflow
	var err error

	switch criteria := req.Criteria.(type) {
	case *workflowv1.GetWorkflowRequest_WorkflowId:
		wf, err = h.service.GetWorkflow(ctx, criteria.WorkflowId)
	case *workflowv1.GetWorkflowRequest_WorkflowName:
		wf, err = h.service.GetWorkflowByName(ctx, criteria.WorkflowName)
	default:
		return nil, status.Error(codes.InvalidArgument, "workflow_id or workflow_name is required")
	}

	if err != nil {
		return nil, status.Errorf(codes.NotFound, "workflow not found: %v", err)
	}

	return h.workflowToProto(wf), nil
}

func (h *Handler) GetWorkflowFull(ctx context.Context, req *workflowv1.GetWorkflowFullRequest) (*workflowv1.WorkflowFull, error) {
	wf, err := h.service.GetWorkflowFull(ctx, req.WorkflowId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "workflow not found: %v", err)
	}

	return &workflowv1.WorkflowFull{
		Workflow:    h.workflowToProto(wf),
		States:      h.statesToProto(wf.States),
		Transitions: h.transitionsToProto(wf.Transitions),
		Stats:       h.getWorkflowStats(ctx, wf.ID),
	}, nil
}

func (h *Handler) ListWorkflows(ctx context.Context, req *workflowv1.ListWorkflowsRequest) (*workflowv1.ListWorkflowsResponse, error) {
	wfs, err := h.service.ListWorkflows(ctx, req.DepartmentId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list workflows: %v", err)
	}

	var pbWorkflows []*workflowv1.Workflow
	for _, wf := range wfs {
		pbWorkflows = append(pbWorkflows, h.workflowToProto(wf))
	}

	return &workflowv1.ListWorkflowsResponse{Workflows: pbWorkflows}, nil
}

func (h *Handler) UpdateWorkflow(ctx context.Context, req *workflowv1.UpdateWorkflowRequest) (*workflowv1.Workflow, error) {
	wf, err := h.service.UpdateWorkflow(ctx, req.Id, &UpdateWorkflowInput{
		Name:        req.Name,
		Description: req.Description,
		Settings:    req.Settings.AsMap(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update workflow: %v", err)
	}
	return h.workflowToProto(wf), nil
}

func (h *Handler) DeleteWorkflow(ctx context.Context, req *workflowv1.DeleteWorkflowRequest) (*emptypb.Empty, error) {
	if err := h.service.DeleteWorkflow(ctx, req.WorkflowId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete workflow: %v", err)
	}
	return &emptypb.Empty{}, nil
}

// ==================== WORKFLOW MANAGEMENT ====================

func (h *Handler) SetActiveWorkflow(ctx context.Context, req *workflowv1.SetActiveWorkflowRequest) (*workflowv1.Workflow, error) {
	wf, err := h.service.SetActiveWorkflow(ctx, req.WorkflowId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set active workflow: %v", err)
	}
	return h.workflowToProto(wf), nil
}

func (h *Handler) GetActiveWorkflowByDepartment(ctx context.Context, req *workflowv1.GetActiveWorkflowByDepartmentRequest) (*workflowv1.Workflow, error) {
	wf, err := h.service.GetActiveWorkflowByDepartment(ctx, req.DepartmentId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "no active workflow found: %v", err)
	}
	return h.workflowToProto(wf), nil
}

func (h *Handler) CloneWorkflow(ctx context.Context, req *workflowv1.CloneWorkflowRequest) (*workflowv1.Workflow, error) {
	wf, err := h.service.CloneWorkflow(ctx, req.SourceWorkflowId, req.TargetDepartmentId, req.NewName, req.CloneAsTemplate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to clone workflow: %v", err)
	}
	return h.workflowToProto(wf), nil
}

func (h *Handler) CreateNewVersion(ctx context.Context, req *workflowv1.CreateNewVersionRequest) (*workflowv1.Workflow, error) {
	wf, err := h.service.CreateNewVersion(ctx, req.WorkflowId, req.Changelog)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create new version: %v", err)
	}
	return h.workflowToProto(wf), nil
}

func (h *Handler) ValidateWorkflow(ctx context.Context, req *workflowv1.ValidateWorkflowRequest) (*workflowv1.ValidateWorkflowResponse, error) {
	result, err := h.service.ValidateWorkflow(ctx, req.WorkflowId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to validate workflow: %v", err)
	}

	var errors []*workflowv1.ValidationError
	for _, e := range result.Errors {
		errors = append(errors, &workflowv1.ValidationError{
			Code:    e.Code,
			Message: e.Message,
			Path:    e.Path,
		})
	}

	var warnings []*workflowv1.ValidationWarning
	for _, w := range result.Warnings {
		warnings = append(warnings, &workflowv1.ValidationWarning{
			Code:    w.Code,
			Message: w.Message,
			Path:    w.Path,
		})
	}

	return &workflowv1.ValidateWorkflowResponse{
		IsValid:  result.IsValid,
		Errors:   errors,
		Warnings: warnings,
	}, nil
}

// ==================== STATE CRUD ====================

func (h *Handler) CreateState(ctx context.Context, req *workflowv1.CreateStateRequest) (*workflowv1.State, error) {
	if req.WorkflowId == 0 {
		return nil, status.Error(codes.InvalidArgument, "workflow_id is required")
	}
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	var config map[string]interface{}
	if req.Config != nil {
		config = req.Config.AsMap()
	}

	state, err := h.service.CreateState(ctx, &CreateStateInput{
		WorkflowID:   req.WorkflowId,
		Name:         req.Name,
		DisplayName:  req.DisplayName,
		Description:  req.Description,
		Type:         req.Type.String(),
		Config:       config,
		DurationDays: req.DurationDays,
		IsInitial:    req.IsInitial,
		IsFinal:      req.IsFinal,
		IsOptional:   req.IsOptional,
		OrderIndex:   req.OrderIndex,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create state: %v", err)
	}

	return h.stateToProto(state), nil
}

func (h *Handler) GetState(ctx context.Context, req *workflowv1.GetStateRequest) (*workflowv1.State, error) {
	state, err := h.service.GetState(ctx, req.StateId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "state not found: %v", err)
	}
	return h.stateToProto(state), nil
}

func (h *Handler) ListStates(ctx context.Context, req *workflowv1.ListStatesRequest) (*workflowv1.ListStatesResponse, error) {
	states, err := h.service.ListStates(ctx, req.WorkflowId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list states: %v", err)
	}
	return &workflowv1.ListStatesResponse{States: h.statesToProto(states)}, nil
}

func (h *Handler) UpdateState(ctx context.Context, req *workflowv1.UpdateStateRequest) (*workflowv1.State, error) {
	var config map[string]interface{}
	if req.Config != nil {
		config = req.Config.AsMap()
	}

	state, err := h.service.UpdateState(ctx, req.Id, &UpdateStateInput{
		Name:         req.Name,
		DisplayName:  req.DisplayName,
		Description:  req.Description,
		Config:       config,
		DurationDays: req.DurationDays,
		IsOptional:   req.IsOptional,
		OrderIndex:   req.OrderIndex,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update state: %v", err)
	}
	return h.stateToProto(state), nil
}

func (h *Handler) DeleteState(ctx context.Context, req *workflowv1.DeleteStateRequest) (*emptypb.Empty, error) {
	if err := h.service.DeleteState(ctx, req.StateId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete state: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) ReorderStates(ctx context.Context, req *workflowv1.ReorderStatesRequest) (*workflowv1.ListStatesResponse, error) {
	if err := h.service.ReorderStates(ctx, req.WorkflowId, req.StateIds); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to reorder states: %v", err)
	}
	states, _ := h.service.ListStates(ctx, req.WorkflowId)
	return &workflowv1.ListStatesResponse{States: h.statesToProto(states)}, nil
}

// ==================== TRANSITION CRUD ====================

func (h *Handler) CreateTransition(ctx context.Context, req *workflowv1.CreateTransitionRequest) (*workflowv1.Transition, error) {
	tr, err := h.service.CreateTransition(ctx, &CreateTransitionInput{
		WorkflowID:  req.WorkflowId,
		EventName:   req.EventName,
		DisplayName: req.DisplayName,
		FromStateID: req.FromStateId,
		ToStateID:   req.ToStateId,
		ButtonLabel: req.ButtonLabel,
		ButtonColor: req.ButtonColor,
		ConfirmText: req.ConfirmText,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create transition: %v", err)
	}
	return h.transitionToProto(tr), nil
}

func (h *Handler) UpdateTransition(ctx context.Context, req *workflowv1.UpdateTransitionRequest) (*workflowv1.Transition, error) {
	tr, err := h.service.UpdateTransition(ctx, req.Id, &UpdateTransitionInput{
		DisplayName: req.DisplayName,
		ButtonLabel: req.ButtonLabel,
		ButtonColor: req.ButtonColor,
		ConfirmText: req.ConfirmText,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update transition: %v", err)
	}
	return h.transitionToProto(tr), nil
}

func (h *Handler) DeleteTransition(ctx context.Context, req *workflowv1.DeleteTransitionRequest) (*emptypb.Empty, error) {
	if err := h.service.DeleteTransition(ctx, req.TransitionId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete transition: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) ListTransitions(ctx context.Context, req *workflowv1.ListTransitionsRequest) (*workflowv1.ListTransitionsResponse, error) {
	transitions, err := h.service.ListTransitions(ctx, req.WorkflowId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list transitions: %v", err)
	}
	return &workflowv1.ListTransitionsResponse{Transitions: h.transitionsToProto(transitions)}, nil
}

// ==================== ACTION CRUD ====================

func (h *Handler) CreateStateAction(ctx context.Context, req *workflowv1.CreateStateActionRequest) (*workflowv1.StateAction, error) {
	var config map[string]interface{}
	if req.Config != nil {
		config = req.Config.AsMap()
	}

	action, err := h.service.CreateStateAction(ctx, &CreateStateActionInput{
		StateID:    req.StateId,
		Name:       req.Name,
		Type:       req.Type.String(),
		Trigger:    req.Trigger.String(),
		OrderIndex: req.OrderIndex,
		Config:     config,
		IsEnabled:  req.IsEnabled,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create action: %v", err)
	}
	return h.actionToProto(action), nil
}

func (h *Handler) UpdateStateAction(ctx context.Context, req *workflowv1.UpdateStateActionRequest) (*workflowv1.StateAction, error) {
	var config map[string]interface{}
	if req.Config != nil {
		config = req.Config.AsMap()
	}

	action, err := h.service.UpdateStateAction(ctx, req.Id, &UpdateStateActionInput{
		Name:       req.Name,
		Config:     config,
		OrderIndex: req.OrderIndex,
		IsEnabled:  req.IsEnabled,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update action: %v", err)
	}
	return h.actionToProto(action), nil
}

func (h *Handler) DeleteStateAction(ctx context.Context, req *workflowv1.DeleteStateActionRequest) (*emptypb.Empty, error) {
	if err := h.service.DeleteStateAction(ctx, req.ActionId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete action: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) ListStateActions(ctx context.Context, req *workflowv1.ListStateActionsRequest) (*workflowv1.ListStateActionsResponse, error) {
	actions, err := h.service.ListStateActions(ctx, req.StateId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list actions: %v", err)
	}
	return &workflowv1.ListStateActionsResponse{Actions: h.actionsToProto(actions)}, nil
}

// ==================== RUNTIME ====================

func (h *Handler) GetNextState(ctx context.Context, req *workflowv1.GetNextStateRequest) (*workflowv1.State, error) {
	state, err := h.service.GetNextState(ctx, req.CurrentStateId, req.EventName)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "next state not found: %v", err)
	}
	return h.stateToProto(state), nil
}

func (h *Handler) GetAvailableTransitions(ctx context.Context, req *workflowv1.GetAvailableTransitionsRequest) (*workflowv1.GetAvailableTransitionsResponse, error) {
	transitions, err := h.service.GetAvailableTransitions(ctx, req.ProjectId, req.CurrentStateId, req.UserId, req.UserRole)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get transitions: %v", err)
	}

	var pbTransitions []*workflowv1.AvailableTransition
	for _, t := range transitions {
		pbTransitions = append(pbTransitions, &workflowv1.AvailableTransition{
			Transition:          h.transitionToProto(t.Transition),
			CanExecute:          t.CanExecute,
			BlockedReason:       t.BlockedReason,
			MissingRequirements: t.MissingRequirements,
		})
	}

	return &workflowv1.GetAvailableTransitionsResponse{Transitions: pbTransitions}, nil
}

func (h *Handler) GetStepConfiguration(ctx context.Context, req *workflowv1.GetStepConfigurationRequest) (*workflowv1.StepConfiguration, error) {
	state, err := h.service.GetState(ctx, req.StateId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "state not found: %v", err)
	}

	config := &workflowv1.StepConfiguration{
		StateId:      state.ID,
		StateName:    state.Name,
		StateType:    h.parseStateType(state.Type),
		DurationDays: state.DurationDays,
	}

	// Parse config JSON
	var configMap map[string]interface{}
	if err := json.Unmarshal(state.Config, &configMap); err == nil {
		if tc, ok := configMap["team_config"].(map[string]interface{}); ok {
			config.TeamConfig = &workflowv1.TeamConfig{
				MinSize:   int32(getIntFromMap(tc, "min_size", 1)),
				MaxSize:   int32(getIntFromMap(tc, "max_size", 3)),
				AllowSolo: getBoolFromMap(tc, "allow_solo", true),
			}
		}
		if fc, ok := configMap["file_config"].(map[string]interface{}); ok {
			config.FileConfig = &workflowv1.FileConfig{
				MaxFiles:     int32(getIntFromMap(fc, "max_files", 1)),
				MaxSizeBytes: int64(getIntFromMap(fc, "max_size_bytes", 10485760)),
			}
		}
	}

	return config, nil
}

func (h *Handler) ExecuteTransition(ctx context.Context, req *workflowv1.ExecuteTransitionRequest) (*workflowv1.ExecuteTransitionResponse, error) {
	var payload map[string]interface{}
	if req.Payload != nil {
		payload = req.Payload.AsMap()
	}

	result, err := h.service.ExecuteTransition(ctx, &ExecuteTransitionInput{
		ProjectID:    req.ProjectId,
		TransitionID: req.TransitionId,
		UserID:       req.UserId,
		Payload:      payload,
	})
	if err != nil {
		return &workflowv1.ExecuteTransitionResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &workflowv1.ExecuteTransitionResponse{
		Success:         true,
		NewStateId:      result.NewStateID,
		NewStateName:    result.NewStateName,
		ExecutedActions: result.ExecutedActions,
	}, nil
}

// ==================== TEMPLATES ====================

func (h *Handler) ListTemplates(ctx context.Context, req *workflowv1.ListTemplatesRequest) (*workflowv1.ListTemplatesResponse, error) {
	templates, err := h.service.ListTemplates(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list templates: %v", err)
	}

	var pbTemplates []*workflowv1.WorkflowTemplate
	for _, t := range templates {
		pbTemplates = append(pbTemplates, &workflowv1.WorkflowTemplate{
			Id:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Category:    t.Category,
		})
	}

	return &workflowv1.ListTemplatesResponse{Templates: pbTemplates}, nil
}

func (h *Handler) CreateFromTemplate(ctx context.Context, req *workflowv1.CreateFromTemplateRequest) (*workflowv1.Workflow, error) {
	wf, err := h.service.CreateFromTemplate(ctx, req.TemplateId, req.DepartmentId, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create from template: %v", err)
	}
	return h.workflowToProto(wf), nil
}

// ==================== HELPERS ====================

func (h *Handler) workflowToProto(wf *Workflow) *workflowv1.Workflow {
	if wf == nil {
		return nil
	}

	var settings *structpb.Struct
	if len(wf.Settings) > 0 {
		var settingsMap map[string]interface{}
		if err := json.Unmarshal(wf.Settings, &settingsMap); err == nil {
			settings, _ = structpb.NewStruct(settingsMap)
		}
	}

	var parentID int64
	if wf.ParentID != nil {
		parentID = *wf.ParentID
	}
	return &workflowv1.Workflow{
		Id:           wf.ID,
		Name:         wf.Name,
		Description:  wf.Description,
		DepartmentId: wf.DepartmentID,
		Version:      wf.Version,
		IsActive:     wf.IsActive,
		IsTemplate:   wf.IsTemplate,
		ParentId:     parentID,
		Settings:     settings,
		States:       h.statesToProto(wf.States),
		Transitions:  h.transitionsToProto(wf.Transitions),
		CreatedAt:    timestamppb.New(wf.CreatedAt),
		UpdatedAt:    timestamppb.New(wf.UpdatedAt),
	}
}

func (h *Handler) stateToProto(s *State) *workflowv1.State {
	if s == nil {
		return nil
	}

	var config *structpb.Struct
	if len(s.Config) > 0 {
		var configMap map[string]interface{}
		if err := json.Unmarshal(s.Config, &configMap); err == nil {
			config, _ = structpb.NewStruct(configMap)
		}
	}

	return &workflowv1.State{
		Id:           s.ID,
		WorkflowId:   s.WorkflowID,
		Name:         s.Name,
		DisplayName:  s.DisplayName,
		Description:  s.Description,
		OrderIndex:   s.OrderIndex,
		Type:         h.parseStateType(s.Type),
		IsInitial:    s.IsInitial,
		IsFinal:      s.IsFinal,
		IsOptional:   s.IsOptional,
		Config:       config,
		DurationDays: s.DurationDays,
		DurationMode: s.DurationMode,
		Color:        s.Color,
		Icon:         s.Icon,
		Actions:      h.actionsToProto(s.Actions),
	}
}

func (h *Handler) statesToProto(states []State) []*workflowv1.State {
	var result []*workflowv1.State
	for _, s := range states {
		result = append(result, h.stateToProto(&s))
	}
	return result
}

func (h *Handler) transitionToProto(t *Transition) *workflowv1.Transition {
	if t == nil {
		return nil
	}
	return &workflowv1.Transition{
		Id:          t.ID,
		WorkflowId:  t.WorkflowID,
		EventName:   t.EventName,
		DisplayName: t.DisplayName,
		FromStateId: t.FromStateID,
		ToStateId:   t.ToStateID,
		ButtonLabel: t.ButtonLabel,
		ButtonColor: t.ButtonColor,
		ConfirmText: t.ConfirmText,
	}
}

func (h *Handler) transitionsToProto(transitions []Transition) []*workflowv1.Transition {
	var result []*workflowv1.Transition
	for _, t := range transitions {
		result = append(result, h.transitionToProto(&t))
	}
	return result
}

func (h *Handler) actionToProto(a *StateAction) *workflowv1.StateAction {
	if a == nil {
		return nil
	}

	var config *structpb.Struct
	if len(a.Config) > 0 {
		var configMap map[string]interface{}
		if err := json.Unmarshal(a.Config, &configMap); err == nil {
			config, _ = structpb.NewStruct(configMap)
		}
	}

	return &workflowv1.StateAction{
		Id:         a.ID,
		StateId:    a.StateID,
		Name:       a.Name,
		Type:       h.parseActionType(a.Type),
		Trigger:    h.parseActionTrigger(a.Trigger),
		OrderIndex: a.OrderIndex,
		Config:     config,
		IsEnabled:  a.IsEnabled,
	}
}

func (h *Handler) actionsToProto(actions []StateAction) []*workflowv1.StateAction {
	var result []*workflowv1.StateAction
	for _, a := range actions {
		result = append(result, h.actionToProto(&a))
	}
	return result
}

func (h *Handler) parseStateType(t string) workflowv1.StateType {
	switch t {
	case "TEAM_FORMATION":
		return workflowv1.StateType_TEAM_FORMATION
	case "SUPERVISOR_SELECTION":
		return workflowv1.StateType_SUPERVISOR_SELECTION
	case "TOPIC_APPROVAL":
		return workflowv1.StateType_TOPIC_APPROVAL
	case "DOCUMENT_UPLOAD":
		return workflowv1.StateType_DOCUMENT_UPLOAD
	case "FORM_SUBMIT":
		return workflowv1.StateType_FORM_SUBMIT
	case "EXTERNAL_CHECK":
		return workflowv1.StateType_EXTERNAL_CHECK
	case "REVIEW":
		return workflowv1.StateType_REVIEW
	case "APPROVAL":
		return workflowv1.StateType_APPROVAL
	case "DEFENSE":
		return workflowv1.StateType_DEFENSE
	case "MILESTONE":
		return workflowv1.StateType_MILESTONE
	case "GRADING":
		return workflowv1.StateType_GRADING
	case "COMPLETED":
		return workflowv1.StateType_COMPLETED
	default:
		return workflowv1.StateType_STATE_TYPE_UNSPECIFIED
	}
}

func (h *Handler) parseActionType(t string) workflowv1.ActionType {
	switch t {
	case "SEND_NOTIFICATION":
		return workflowv1.ActionType_SEND_NOTIFICATION
	case "SEND_EMAIL":
		return workflowv1.ActionType_SEND_EMAIL
	case "ASSIGN_TASK":
		return workflowv1.ActionType_ASSIGN_TASK
	case "CALL_WEBHOOK":
		return workflowv1.ActionType_CALL_WEBHOOK
	default:
		return workflowv1.ActionType_ACTION_TYPE_UNSPECIFIED
	}
}

func (h *Handler) parseActionTrigger(t string) workflowv1.ActionTrigger {
	switch t {
	case "ON_ENTER":
		return workflowv1.ActionTrigger_ON_ENTER
	case "ON_EXIT":
		return workflowv1.ActionTrigger_ON_EXIT
	case "ON_DEADLINE":
		return workflowv1.ActionTrigger_ON_DEADLINE
	default:
		return workflowv1.ActionTrigger_TRIGGER_UNSPECIFIED
	}
}

func (h *Handler) getWorkflowStats(ctx context.Context, workflowID int64) *workflowv1.WorkflowStats {
	// TODO: implement stats query
	return &workflowv1.WorkflowStats{}
}

func getIntFromMap(m map[string]interface{}, key string, defaultVal int) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return defaultVal
}

func getBoolFromMap(m map[string]interface{}, key string, defaultVal bool) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return defaultVal
}
