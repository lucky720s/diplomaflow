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

type Handler struct {
	workflowv1.UnimplementedWorkflowServiceServer
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) CreateWorkflow(ctx context.Context, req *workflowv1.CreateWorkflowRequest) (*workflowv1.Workflow, error) {
	wf, err := h.repo.CreateWorkflow(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create workflow: %v", err)
	}
	return toProtoWorkflow(wf), nil
}

func (h *Handler) CreateState(ctx context.Context, req *workflowv1.CreateStateRequest) (*workflowv1.State, error) {
	st, err := h.repo.CreateState(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create state: %v", err)
	}
	return toProtoState(st), nil
}
func (h *Handler) GetState(ctx context.Context, req *workflowv1.GetStateRequest) (*workflowv1.State, error) {
	st, err := h.repo.GetState(ctx, req.GetStateId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "state not found: %v", err)
	}
	return toProtoState(st), nil
}

func (h *Handler) CreateTransition(ctx context.Context, req *workflowv1.CreateTransitionRequest) (*workflowv1.Transition, error) {
	tr, err := h.repo.CreateTransition(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create transition: %v", err)
	}
	return toProtoTransition(tr), nil
}
func (h *Handler) DeleteTransition(ctx context.Context, req *workflowv1.DeleteTransitionRequest) (*emptypb.Empty, error) {
	if err := h.repo.DeleteTransition(ctx, req.GetId()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete transition: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) CreateStateAction(ctx context.Context, req *workflowv1.CreateStateActionRequest) (*workflowv1.StateAction, error) {
	sa, err := h.repo.CreateStateAction(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create action: %v", err)
	}
	return toProtoStateAction(sa), nil
}
func (h *Handler) ListStateActions(ctx context.Context, req *workflowv1.ListStateActionsRequest) (*workflowv1.ListStateActionsResponse, error) {
	actions, err := h.repo.ListStateActions(ctx, req.GetStateId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list actions: %v", err)
	}
	protoActions := make([]*workflowv1.StateAction, len(actions))
	for i, action := range actions {
		protoActions[i] = toProtoStateAction(action)
	}
	return &workflowv1.ListStateActionsResponse{Actions: protoActions}, nil
}

func (h *Handler) GetNextState(ctx context.Context, req *workflowv1.GetNextStateRequest) (*workflowv1.State, error) {
	state, err := h.repo.GetNextState(ctx, req.GetCurrentStateId(), req.GetEventName())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "no transition found: %v", err)
	}
	return toProtoState(state), nil
}

func toProtoWorkflow(wf *Workflow) *workflowv1.Workflow {
	return &workflowv1.Workflow{Id: wf.ID, Name: wf.Name, DepartmentId: wf.DepartmentID}
}
func toProtoState(st *State) *workflowv1.State {
	cfg, _ := structpb.NewStruct(nil)
	_ = json.Unmarshal(st.Config, &cfg)
	return &workflowv1.State{Id: st.ID, WorkflowId: st.WorkflowID, Name: st.Name, Description: st.Description, Type: workflowv1.StateType(workflowv1.StateType_value[st.Type]), Config: cfg, DurationDays: st.DurationDays}
}
func toProtoTransition(tr *Transition) *workflowv1.Transition {
	return &workflowv1.Transition{Id: tr.ID, WorkflowId: tr.WorkflowID, EventName: tr.EventName, FromStateId: tr.FromStateID, ToStateId: tr.ToStateID}
}
func toProtoStateAction(sa *StateAction) *workflowv1.StateAction {
	cfg, _ := structpb.NewStruct(nil)
	_ = json.Unmarshal(sa.Config, &cfg)
	return &workflowv1.StateAction{Id: sa.ID, StateId: sa.StateID, Type: workflowv1.StateAction_ActionType(workflowv1.StateAction_ActionType_value[sa.Type]), Trigger: workflowv1.StateAction_Trigger(workflowv1.StateAction_Trigger_value[sa.Trigger]), Config: cfg}
}
