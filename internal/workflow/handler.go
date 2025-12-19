package workflow

import (
	"context"
	"encoding/json"

	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

type WorkflowUseCase interface {
	CreateWorkflow(ctx context.Context, name string, departmentID int64) (*Workflow, error)
	GetWorkflow(ctx context.Context, id int64) (*Workflow, error)
	GetWorkflowByName(ctx context.Context, name string) (*Workflow, error)
	ListWorkflows(ctx context.Context, departmentID int64) ([]*Workflow, error)
	UpdateWorkflow(ctx context.Context, id int64, name string) (*Workflow, error)
	DeleteWorkflow(ctx context.Context, id int64) error

	CreateState(ctx context.Context, req *workflowv1.CreateStateRequest) (*State, error)
	GetState(ctx context.Context, id int64) (*State, error)
	ListStates(ctx context.Context, workflowID int64) ([]*State, error)
	UpdateState(ctx context.Context, req *workflowv1.UpdateStateRequest) (*State, error)
	DeleteState(ctx context.Context, id int64) error

	CreateTransition(ctx context.Context, req *workflowv1.CreateTransitionRequest) (*Transition, error)
	DeleteTransition(ctx context.Context, id int64) error

	CreateStateAction(ctx context.Context, req *workflowv1.CreateStateActionRequest) (*StateAction, error)
	ListStateActions(ctx context.Context, stateID int64) ([]*StateAction, error)
	DeleteStateAction(ctx context.Context, id int64) error

	SetActiveWorkflow(ctx context.Context, workflowID int64) (*Workflow, error)
	GetActiveWorkflowByDepartment(ctx context.Context, departmentID int64) (*Workflow, error)
	GetNextState(ctx context.Context, currentStateID int64, eventName string) (*State, error)
}

type Handler struct {
	workflowv1.UnimplementedWorkflowServiceServer
	service WorkflowUseCase
}

func NewHandler(service WorkflowUseCase) *Handler {
	return &Handler{service: service}
}

func toProtoWorkflow(wf *Workflow) *workflowv1.Workflow {
	if wf == nil {
		return nil
	}

	pbWorkflow := &workflowv1.Workflow{
		Id:           wf.ID,
		Name:         wf.Name,
		DepartmentId: wf.DepartmentID,
		IsActive:     wf.IsActive,
	}
	for _, step := range wf.Steps {
		pbWorkflow.Steps = append(pbWorkflow.Steps, toProtoState(&step))
	}

	return pbWorkflow
}

func toProtoState(st *State) *workflowv1.State {
	if st == nil {
		return nil
	}
	var configMap map[string]interface{}
	_ = json.Unmarshal(st.Config, &configMap)
	configStruct, _ := structpb.NewStruct(configMap)

	return &workflowv1.State{
		Id:           st.ID,
		WorkflowId:   st.WorkflowID,
		Name:         st.Name,
		Description:  st.Description,
		Type:         workflowv1.StateType(workflowv1.StateType_value[st.Type]),
		Config:       configStruct,
		DurationDays: st.DurationDays,
	}
}

func toProtoTransition(tr *Transition) *workflowv1.Transition {
	if tr == nil {
		return nil
	}
	return &workflowv1.Transition{
		Id:          tr.ID,
		WorkflowId:  tr.WorkflowID,
		EventName:   tr.EventName,
		FromStateId: tr.FromStateID,
		ToStateId:   tr.ToStateID,
	}
}

func toProtoStateAction(sa *StateAction) *workflowv1.StateAction {
	if sa == nil {
		return nil
	}
	var configMap map[string]interface{}
	_ = json.Unmarshal(sa.Config, &configMap)
	configStruct, _ := structpb.NewStruct(configMap)

	return &workflowv1.StateAction{
		Id:      sa.ID,
		StateId: sa.StateID,
		Type:    workflowv1.StateAction_ActionType(workflowv1.StateAction_ActionType_value[sa.Type]),
		Trigger: workflowv1.StateAction_Trigger(workflowv1.StateAction_Trigger_value[sa.Trigger]),
		Config:  configStruct,
	}
}

func (h *Handler) CreateWorkflow(ctx context.Context, req *workflowv1.CreateWorkflowRequest) (*workflowv1.Workflow, error) {
	wf, err := h.service.CreateWorkflow(ctx, req.Name, req.DepartmentId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create workflow: %v", err)
	}
	return toProtoWorkflow(wf), nil
}

func (h *Handler) GetWorkflow(ctx context.Context, req *workflowv1.GetWorkflowRequest) (*workflowv1.Workflow, error) {
	var wf *Workflow
	var err error

	switch criteria := req.Criteria.(type) {
	case *workflowv1.GetWorkflowRequest_WorkflowId:
		wf, err = h.service.GetWorkflow(ctx, criteria.WorkflowId)
	case *workflowv1.GetWorkflowRequest_Name:
		wf, err = h.service.GetWorkflowByName(ctx, criteria.Name)
	default:
		return nil, status.Error(codes.InvalidArgument, "workflow_id or name must be provided")
	}

	if err != nil {
		return nil, status.Errorf(codes.NotFound, "workflow not found: %v", err)
	}

	return toProtoWorkflow(wf), nil
}
func (h *Handler) ListWorkflows(ctx context.Context, req *workflowv1.ListWorkflowsRequest) (*workflowv1.ListWorkflowsResponse, error) {
	wfs, err := h.service.ListWorkflows(ctx, req.DepartmentId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list workflows: %v", err)
	}
	var pbWfs []*workflowv1.Workflow
	for _, wf := range wfs {
		pbWfs = append(pbWfs, toProtoWorkflow(wf))
	}
	return &workflowv1.ListWorkflowsResponse{Workflows: pbWfs}, nil
}

func (h *Handler) UpdateWorkflow(ctx context.Context, req *workflowv1.UpdateWorkflowRequest) (*workflowv1.Workflow, error) {
	wf, err := h.service.UpdateWorkflow(ctx, req.Id, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update workflow: %v", err)
	}
	return toProtoWorkflow(wf), nil
}

func (h *Handler) DeleteWorkflow(ctx context.Context, req *workflowv1.DeleteWorkflowRequest) (*emptypb.Empty, error) {
	if err := h.service.DeleteWorkflow(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete workflow: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) CreateState(ctx context.Context, req *workflowv1.CreateStateRequest) (*workflowv1.State, error) {
	st, err := h.service.CreateState(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create state: %v", err)
	}
	return toProtoState(st), nil
}

func (h *Handler) GetState(ctx context.Context, req *workflowv1.GetStateRequest) (*workflowv1.State, error) {
	st, err := h.service.GetState(ctx, req.StateId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "state not found")
	}
	return toProtoState(st), nil
}

func (h *Handler) ListStates(ctx context.Context, req *workflowv1.ListStatesRequest) (*workflowv1.ListStatesResponse, error) {
	states, err := h.service.ListStates(ctx, req.WorkflowId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list states: %v", err)
	}
	var pbStates []*workflowv1.State
	for _, st := range states {
		pbStates = append(pbStates, toProtoState(st))
	}
	return &workflowv1.ListStatesResponse{States: pbStates}, nil
}

func (h *Handler) UpdateState(ctx context.Context, req *workflowv1.UpdateStateRequest) (*workflowv1.State, error) {
	st, err := h.service.UpdateState(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update state: %v", err)
	}
	return toProtoState(st), nil
}

func (h *Handler) DeleteState(ctx context.Context, req *workflowv1.DeleteStateRequest) (*emptypb.Empty, error) {
	if err := h.service.DeleteState(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete state: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) CreateTransition(ctx context.Context, req *workflowv1.CreateTransitionRequest) (*workflowv1.Transition, error) {
	tr, err := h.service.CreateTransition(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create transition: %v", err)
	}
	return toProtoTransition(tr), nil
}

func (h *Handler) DeleteTransition(ctx context.Context, req *workflowv1.DeleteTransitionRequest) (*emptypb.Empty, error) {
	if err := h.service.DeleteTransition(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete transition: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) CreateStateAction(ctx context.Context, req *workflowv1.CreateStateActionRequest) (*workflowv1.StateAction, error) {
	sa, err := h.service.CreateStateAction(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create action: %v", err)
	}
	return toProtoStateAction(sa), nil
}

func (h *Handler) ListStateActions(ctx context.Context, req *workflowv1.ListStateActionsRequest) (*workflowv1.ListStateActionsResponse, error) {
	actions, err := h.service.ListStateActions(ctx, req.StateId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list actions: %v", err)
	}
	var pbActions []*workflowv1.StateAction
	for _, sa := range actions {
		pbActions = append(pbActions, toProtoStateAction(sa))
	}
	return &workflowv1.ListStateActionsResponse{Actions: pbActions}, nil
}

func (h *Handler) DeleteStateAction(ctx context.Context, req *workflowv1.DeleteStateActionRequest) (*emptypb.Empty, error) {
	if err := h.service.DeleteStateAction(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete action: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) SetActiveWorkflow(ctx context.Context, req *workflowv1.SetActiveWorkflowRequest) (*workflowv1.Workflow, error) {
	wf, err := h.service.SetActiveWorkflow(ctx, req.WorkflowId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set active workflow: %v", err)
	}
	return toProtoWorkflow(wf), nil
}

func (h *Handler) GetActiveWorkflowByDepartment(ctx context.Context, req *workflowv1.GetActiveWorkflowByDepartmentRequest) (*workflowv1.Workflow, error) {
	wf, err := h.service.GetActiveWorkflowByDepartment(ctx, req.DepartmentId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "no active workflow found")
	}
	return toProtoWorkflow(wf), nil
}

func (h *Handler) GetNextState(ctx context.Context, req *workflowv1.GetNextStateRequest) (*workflowv1.State, error) {
	st, err := h.service.GetNextState(ctx, req.CurrentStateId, req.EventName)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "next state not found")
	}
	return toProtoState(st), nil
}
func (h *Handler) GetStepConfiguration(ctx context.Context, req *workflowv1.GetStepConfigurationRequest) (*workflowv1.StepConfiguration, error) {
	state, err := h.service.GetState(ctx, req.StateId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "state not found")
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal(state.Config, &configMap); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to parse config")
	}

	config := &workflowv1.StepConfiguration{}
	if tc, ok := configMap["team_config"].(map[string]interface{}); ok {
		config.TeamConfig = &workflowv1.TeamConfig{}
		if v, ok := tc["min_size"].(float64); ok {
			config.TeamConfig.MinSize = int32(v)
		}
		if v, ok := tc["max_size"].(float64); ok {
			config.TeamConfig.MaxSize = int32(v)
		}
		if v, ok := tc["allow_solo"].(bool); ok {
			config.TeamConfig.AllowSolo = v
		}
	}
	if fr, ok := configMap["file_requirements"].(map[string]interface{}); ok {
		config.FileRequirements = &workflowv1.FileRequirements{}
		if v, ok := fr["max_files"].(float64); ok {
			config.FileRequirements.MaxFiles = int32(v)
		}
		if v, ok := fr["max_size_bytes"].(float64); ok {
			config.FileRequirements.MaxSizeBytes = int64(v)
		}
		if exts, ok := fr["allowed_extensions"].([]interface{}); ok {
			for _, ext := range exts {
				if s, ok := ext.(string); ok {
					config.FileRequirements.AllowedExtensions = append(config.FileRequirements.AllowedExtensions, s)
				}
			}
		}
	}
	if roles, ok := configMap["allowed_roles"].([]interface{}); ok {
		for _, r := range roles {
			if s, ok := r.(string); ok {
				config.AllowedRoles = append(config.AllowedRoles, s)
			}
		}
	}

	return config, nil
}
