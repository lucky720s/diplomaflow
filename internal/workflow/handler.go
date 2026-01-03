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
	var settings map[string]interface{}
	if req.Settings != nil {
		settings = req.Settings.AsMap()
	}

	wf, err := h.service.UpdateWorkflow(ctx, req.Id, &UpdateWorkflowInput{
		Name:        req.Name,
		Description: req.Description,
		Settings:    settings,
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
		return nil, status.Errorf(codes.NotFound, "no active workflow for department: %v", err)
	}
	return h.workflowToProto(wf), nil
}

func (h *Handler) CloneWorkflow(ctx context.Context, req *workflowv1.CloneWorkflowRequest) (*workflowv1.Workflow, error) {
	wf, err := h.service.CloneWorkflow(ctx, req.SourceWorkflowId, req.TargetDepartmentId, req.NewName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to clone workflow: %v", err)
	}
	return h.workflowToProto(wf), nil
}

func (h *Handler) CreateNewVersion(ctx context.Context, req *workflowv1.CreateNewVersionRequest) (*workflowv1.Workflow, error) {
	wf, err := h.service.CreateNewVersion(ctx, req.WorkflowId)
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

	resp := &workflowv1.ValidateWorkflowResponse{
		IsValid: result.IsValid,
	}

	for _, e := range result.Errors {
		resp.Errors = append(resp.Errors, &workflowv1.ValidationError{
			Code:    e.Code,
			Message: e.Message,
			Path:    e.Path,
		})
	}

	for _, w := range result.Warnings {
		resp.Warnings = append(resp.Warnings, &workflowv1.ValidationWarning{
			Code:    w.Code,
			Message: w.Message,
			Path:    w.Path,
		})
	}

	return resp, nil
}

// ==================== STATE CRUD ====================

func (h *Handler) CreateState(ctx context.Context, req *workflowv1.CreateStateRequest) (*workflowv1.State, error) {
	var config map[string]interface{}
	if req.Config != nil {
		config = req.Config.AsMap()
	}

	state, err := h.service.CreateState(ctx, &CreateStateInput{
		WorkflowID:   req.WorkflowId,
		Name:         req.Name,
		DisplayName:  req.DisplayName,
		Description:  req.Description,
		OrderIndex:   req.OrderIndex,
		Type:         req.Type.String(),
		IsInitial:    req.IsInitial,
		IsFinal:      req.IsFinal,
		IsOptional:   req.IsOptional,
		Config:       config,
		DurationDays: req.DurationDays,
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

	return &workflowv1.ListStatesResponse{
		States: h.statesToProto(states),
	}, nil
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
		OrderIndex:   req.OrderIndex,
		IsOptional:   req.IsOptional,
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
	if err := h.service.repo.ReorderStates(ctx, req.WorkflowId, req.StateIds); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to reorder states: %v", err)
	}

	states, err := h.service.ListStates(ctx, req.WorkflowId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list states: %v", err)
	}

	return &workflowv1.ListStatesResponse{
		States: h.statesToProto(states),
	}, nil
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

	return &workflowv1.ListTransitionsResponse{
		Transitions: h.transitionsToProto(transitions),
	}, nil
}

// ==================== STATE ACTIONS ====================

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
		return nil, status.Errorf(codes.Internal, "failed to create state action: %v", err)
	}

	return h.stateActionToProto(action), nil
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
		return nil, status.Errorf(codes.Internal, "failed to update state action: %v", err)
	}

	return h.stateActionToProto(action), nil
}

func (h *Handler) DeleteStateAction(ctx context.Context, req *workflowv1.DeleteStateActionRequest) (*emptypb.Empty, error) {
	if err := h.service.DeleteStateAction(ctx, req.ActionId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete state action: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) ListStateActions(ctx context.Context, req *workflowv1.ListStateActionsRequest) (*workflowv1.ListStateActionsResponse, error) {
	actions, err := h.service.ListStateActions(ctx, req.StateId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list state actions: %v", err)
	}

	var pbActions []*workflowv1.StateAction
	for _, a := range actions {
		pbActions = append(pbActions, h.stateActionToProto(&a))
	}

	return &workflowv1.ListStateActionsResponse{Actions: pbActions}, nil
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
	// TODO: Integrate with engine
	transitions, err := h.service.repo.GetTransitionsFromState(ctx, req.CurrentStateId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get transitions: %v", err)
	}

	var pbTransitions []*workflowv1.AvailableTransition
	for _, tr := range transitions {
		pbTransitions = append(pbTransitions, &workflowv1.AvailableTransition{
			Transition: h.transitionToProto(&tr),
			CanExecute: true, // TODO: Check conditions
		})
	}

	return &workflowv1.GetAvailableTransitionsResponse{
		Transitions: pbTransitions,
	}, nil
}

func (h *Handler) GetStepConfiguration(ctx context.Context, req *workflowv1.GetStepConfigurationRequest) (*workflowv1.StepConfiguration, error) {
	result, err := h.service.GetStepConfiguration(ctx, req.StateId, req.ProjectId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "step configuration not found: %v", err)
	}

	resp := &workflowv1.StepConfiguration{
		StateId:      result.StateID,
		StateName:    result.StateName,
		StateType:    workflowv1.StateType(workflowv1.StateType_value[result.StateType]),
		DurationDays: result.DurationDays,
	}

	if result.TeamConfig != nil {
		resp.TeamConfig = &workflowv1.TeamConfig{
			MinSize:       result.TeamConfig.MinSize,
			MaxSize:       result.TeamConfig.MaxSize,
			AllowSolo:     result.TeamConfig.AllowSolo,
			RequireLeader: result.TeamConfig.RequireLeader,
		}
	}

	if result.FileConfig != nil {
		resp.FileConfig = &workflowv1.FileConfig{
			MaxFiles:          result.FileConfig.MaxFiles,
			MaxSizeBytes:      result.FileConfig.MaxSizeBytes,
			AllowedExtensions: result.FileConfig.AllowedExtensions,
		}
	}

	if result.FormConfig != nil {
		resp.FormConfig = &workflowv1.FormConfig{
			FormId: result.FormConfig.FormID,
		}
		if result.FormConfig.Schema != nil {
			schemaJSON, _ := json.Marshal(result.FormConfig.Schema)
			resp.FormConfig.JsonSchema, _ = structpb.NewStruct(map[string]interface{}{
				"schema": string(schemaJSON),
			})
		}
	}

	if result.ReviewConfig != nil {
		resp.ReviewConfig = &workflowv1.ReviewConfig{
			ReviewerRoles:  result.ReviewConfig.ReviewerRoles,
			MinReviewers:   result.ReviewConfig.MinReviewers,
			RequireComment: result.ReviewConfig.RequireComment,
		}
	}

	return resp, nil
}

func (h *Handler) ExecuteTransition(ctx context.Context, req *workflowv1.ExecuteTransitionRequest) (*workflowv1.ExecuteTransitionResponse, error) {
	// TODO: Integrate with engine
	return nil, status.Error(codes.Unimplemented, "not implemented - use engine directly")
}

// ==================== CONVERTERS ====================

func (h *Handler) workflowToProto(wf *Workflow) *workflowv1.Workflow {
	if wf == nil {
		return nil
	}

	pb := &workflowv1.Workflow{
		Id:           wf.ID,
		Name:         wf.Name,
		Description:  wf.Description,
		DepartmentId: wf.DepartmentID,
		Version:      wf.Version,
		IsActive:     wf.IsActive,
		IsTemplate:   wf.IsTemplate,
		CreatedAt:    timestamppb.New(wf.CreatedAt),
		UpdatedAt:    timestamppb.New(wf.UpdatedAt),
	}

	if wf.ParentID != nil {
		pb.ParentId = *wf.ParentID
	}

	if len(wf.Settings) > 0 {
		var settings map[string]interface{}
		if err := json.Unmarshal(wf.Settings, &settings); err == nil {
			pb.Settings, _ = structpb.NewStruct(settings)
		}
	}

	pb.States = h.statesToProto(wf.States)
	pb.Transitions = h.transitionsToProto(wf.Transitions)

	return pb
}

func (h *Handler) stateToProto(s *State) *workflowv1.State {
	if s == nil {
		return nil
	}

	pb := &workflowv1.State{
		Id:           s.ID,
		WorkflowId:   s.WorkflowID,
		Name:         s.Name,
		DisplayName:  s.DisplayName,
		Description:  s.Description,
		OrderIndex:   s.OrderIndex,
		Type:         workflowv1.StateType(workflowv1.StateType_value[s.Type]),
		IsInitial:    s.IsInitial,
		IsFinal:      s.IsFinal,
		IsOptional:   s.IsOptional,
		DurationDays: s.DurationDays,
		DurationMode: s.DurationMode,
		Color:        s.Color,
		Icon:         s.Icon,
	}

	if len(s.Config) > 0 {
		var config map[string]interface{}
		if err := json.Unmarshal(s.Config, &config); err == nil {
			pb.Config, _ = structpb.NewStruct(config)
		}
	}

	if s.FixedDeadline != nil {
		pb.FixedDeadline = timestamppb.New(*s.FixedDeadline)
	}

	for _, a := range s.Actions {
		pb.Actions = append(pb.Actions, h.stateActionToProto(&a))
	}

	return pb
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

	pb := &workflowv1.Transition{
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

	if len(t.Conditions) > 0 {
		var conditions []map[string]interface{}
		if err := json.Unmarshal(t.Conditions, &conditions); err == nil {
			for _, c := range conditions {
				cond := &workflowv1.TransitionCondition{}
				if v, ok := c["type"].(string); ok {
					cond.Type = v
				}
				if v, ok := c["field"].(string); ok {
					cond.Field = v
				}
				if v, ok := c["operator"].(string); ok {
					cond.Operator = v
				}
				pb.Conditions = append(pb.Conditions, cond)
			}
		}
	}

	return pb
}

func (h *Handler) transitionsToProto(transitions []Transition) []*workflowv1.Transition {
	var result []*workflowv1.Transition
	for _, t := range transitions {
		result = append(result, h.transitionToProto(&t))
	}
	return result
}

func (h *Handler) stateActionToProto(a *StateAction) *workflowv1.StateAction {
	if a == nil {
		return nil
	}

	pb := &workflowv1.StateAction{
		Id:         a.ID,
		StateId:    a.StateID,
		Name:       a.Name,
		Type:       workflowv1.ActionType(workflowv1.ActionType_value[a.Type]),
		Trigger:    workflowv1.ActionTrigger(workflowv1.ActionTrigger_value[a.Trigger]),
		OrderIndex: a.OrderIndex,
		IsEnabled:  a.IsEnabled,
	}

	if len(a.Config) > 0 {
		var config map[string]interface{}
		if err := json.Unmarshal(a.Config, &config); err == nil {
			pb.Config, _ = structpb.NewStruct(config)
		}
	}

	return pb
}

func (h *Handler) GetTeamConfiguration(ctx context.Context, req *workflowv1.GetTeamConfigurationRequest) (*workflowv1.TeamConfigurationResponse, error) {
	result, err := h.service.GetTeamConfiguration(ctx, req.DepartmentId, req.WorkflowId, req.StateId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get team configuration: %v", err)
	}

	resp := &workflowv1.TeamConfigurationResponse{
		WorkflowId:       result.WorkflowID,
		StateId:          result.StateID,
		StateName:        result.StateName,
		InviteExpireDays: result.InviteExpireDays,
		TeamStepRequired: result.TeamStepRequired,
	}

	if result.TeamConfig != nil {
		resp.TeamConfig = &workflowv1.TeamConfig{
			MinSize:       result.TeamConfig.MinSize,
			MaxSize:       result.TeamConfig.MaxSize,
			AllowSolo:     result.TeamConfig.AllowSolo,
			RequireLeader: result.TeamConfig.RequireLeader,
		}
	}

	return resp, nil
}

// CanExecuteTransition проверяет возможность выполнения перехода
func (h *Handler) CanExecuteTransition(ctx context.Context, req *workflowv1.CanExecuteTransitionRequest) (*workflowv1.CanExecuteTransitionResponse, error) {
	// TODO: Implement with engine integration
	return &workflowv1.CanExecuteTransitionResponse{
		CanExecute:    true,
		BlockedReason: "",
	}, nil
}

// GetDepartmentWorkflowConfig возвращает полную конфигурацию workflow для кафедры
func (h *Handler) GetDepartmentWorkflowConfig(ctx context.Context, req *workflowv1.GetDepartmentWorkflowConfigRequest) (*workflowv1.DepartmentWorkflowConfigResponse, error) {
	config, err := h.service.GetDepartmentWorkflowConfiguration(ctx, req.DepartmentId, req.AcademicYear)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get department workflow config: %v", err)
	}

	resp := &workflowv1.DepartmentWorkflowConfigResponse{
		WorkflowId:   config.WorkflowID,
		WorkflowName: config.WorkflowName,
	}

	if config.TeamConfig != nil {
		resp.TeamConfig = &workflowv1.TeamConfig{
			MinSize:   config.TeamConfig.MinSize,
			MaxSize:   config.TeamConfig.MaxSize,
			AllowSolo: config.TeamConfig.AllowSolo,
		}
	}

	for _, s := range config.States {
		sc := &workflowv1.StateConfig{
			Id:           s.ID,
			Name:         s.Name,
			DisplayName:  s.DisplayName,
			OrderIndex:   s.OrderIndex,
			DurationDays: s.DurationDays,
		}
		if s.Deadline != nil {
			sc.Deadline = timestamppb.New(*s.Deadline)
		}
		resp.States = append(resp.States, sc)
	}

	return resp, nil
}
