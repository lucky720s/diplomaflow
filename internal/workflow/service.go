package workflow

import (
	"context"
	"encoding/json"

	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"gorm.io/datatypes"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateWorkflow(ctx context.Context, name string, departmentID int64) (*Workflow, error) {
	wf := &Workflow{
		Name:         name,
		DepartmentID: departmentID,
		IsActive:     false,
	}
	if err := s.repo.CreateWorkflow(ctx, wf); err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *Service) GetWorkflow(ctx context.Context, id int64) (*Workflow, error) {
	return s.repo.GetWorkflow(ctx, id)
}
func (s *Service) GetWorkflowByName(ctx context.Context, name string) (*Workflow, error) {
	return s.repo.GetWorkflowByName(ctx, name)
}
func (s *Service) ListWorkflows(ctx context.Context, departmentID int64) ([]*Workflow, error) {
	return s.repo.ListWorkflows(ctx, departmentID)
}

func (s *Service) UpdateWorkflow(ctx context.Context, id int64, name string) (*Workflow, error) {
	wf, err := s.repo.GetWorkflow(ctx, id)
	if err != nil {
		return nil, err
	}
	wf.Name = name
	if err := s.repo.UpdateWorkflow(ctx, wf); err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *Service) DeleteWorkflow(ctx context.Context, id int64) error {
	return s.repo.DeleteWorkflow(ctx, id)
}

func (s *Service) CreateState(ctx context.Context, req *workflowv1.CreateStateRequest) (*State, error) {
	configBytes, err := req.Config.MarshalJSON()
	if err != nil {
		return nil, err
	}

	st := &State{
		WorkflowID:   req.WorkflowId,
		Name:         req.Name,
		Description:  req.Description,
		Type:         req.Type.String(),
		Config:       datatypes.JSON(configBytes),
		DurationDays: req.DurationDays,
	}
	if err := s.repo.CreateState(ctx, st); err != nil {
		return nil, err
	}
	return st, nil
}

func (s *Service) GetState(ctx context.Context, id int64) (*State, error) {
	return s.repo.GetState(ctx, id)
}

func (s *Service) ListStates(ctx context.Context, workflowID int64) ([]*State, error) {
	return s.repo.ListStates(ctx, workflowID)
}

func (s *Service) UpdateState(ctx context.Context, req *workflowv1.UpdateStateRequest) (*State, error) {
	st, err := s.repo.GetState(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		st.Name = req.Name
	}
	if req.Description != "" {
		st.Description = req.Description
	}
	if req.DurationDays != 0 {
		st.DurationDays = req.DurationDays
	}
	if req.Config != nil {
		configBytes, _ := req.Config.MarshalJSON()
		st.Config = datatypes.JSON(configBytes)
	}

	if err := s.repo.UpdateState(ctx, st); err != nil {
		return nil, err
	}
	return st, nil
}

func (s *Service) DeleteState(ctx context.Context, id int64) error {
	return s.repo.DeleteState(ctx, id)
}

func (s *Service) CreateTransition(ctx context.Context, req *workflowv1.CreateTransitionRequest) (*Transition, error) {
	tr := &Transition{
		WorkflowID:  req.WorkflowId,
		EventName:   req.EventName,
		FromStateID: req.FromStateId,
		ToStateID:   req.ToStateId,
	}
	if err := s.repo.CreateTransition(ctx, tr); err != nil {
		return nil, err
	}
	return tr, nil
}

func (s *Service) DeleteTransition(ctx context.Context, id int64) error {
	return s.repo.DeleteTransition(ctx, id)
}

func (s *Service) CreateStateAction(ctx context.Context, req *workflowv1.CreateStateActionRequest) (*StateAction, error) {
	configBytes, err := req.Config.MarshalJSON()
	if err != nil {
		return nil, err
	}
	sa := &StateAction{
		StateID: req.StateId,
		Type:    req.Type.String(),
		Trigger: req.Trigger.String(),
		Config:  datatypes.JSON(configBytes),
	}
	if err := s.repo.CreateStateAction(ctx, sa); err != nil {
		return nil, err
	}
	return sa, nil
}

func (s *Service) ListStateActions(ctx context.Context, stateID int64) ([]*StateAction, error) {
	return s.repo.ListStateActions(ctx, stateID)
}

func (s *Service) DeleteStateAction(ctx context.Context, id int64) error {
	return s.repo.DeleteStateAction(ctx, id)
}

func (s *Service) SetActiveWorkflow(ctx context.Context, workflowID int64) (*Workflow, error) {
	return s.repo.SetActiveWorkflow(ctx, workflowID)
}

func (s *Service) GetActiveWorkflowByDepartment(ctx context.Context, departmentID int64) (*Workflow, error) {
	return s.repo.GetActiveWorkflowByDepartment(ctx, departmentID)
}

func (s *Service) GetNextState(ctx context.Context, currentStateID int64, eventName string) (*State, error) {
	return s.repo.GetNextState(ctx, currentStateID, eventName)
}

func UnmarshalConfig(data []byte) map[string]interface{} {
	var m map[string]interface{}
	_ = json.Unmarshal(data, &m)
	return m
}
