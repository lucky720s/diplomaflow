package tests_workflow

import (
	"context"

	"github.com/lucky720s/diplomaflow/internal/workflow"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"github.com/stretchr/testify/mock"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateWorkflow(ctx context.Context, wf *workflow.Workflow) error {
	args := m.Called(ctx, wf)
	if wf.ID == 0 {
		wf.ID = 100
	}
	return args.Error(0)
}
func (m *MockRepository) GetWorkflow(ctx context.Context, id int64) (*workflow.Workflow, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.Workflow), args.Error(1)
}
func (m *MockRepository) GetWorkflowByName(ctx context.Context, name string) (*workflow.Workflow, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.Workflow), args.Error(1)
}
func (m *MockRepository) ListWorkflows(ctx context.Context, depID int64) ([]*workflow.Workflow, error) {
	args := m.Called(ctx, depID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*workflow.Workflow), args.Error(1)
}
func (m *MockRepository) UpdateWorkflow(ctx context.Context, wf *workflow.Workflow) error {
	return m.Called(ctx, wf).Error(0)
}
func (m *MockRepository) DeleteWorkflow(ctx context.Context, id int64) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockRepository) CreateState(ctx context.Context, st *workflow.State) error {
	args := m.Called(ctx, st)
	if st.ID == 0 {
		st.ID = 200
	}
	return args.Error(0)
}
func (m *MockRepository) GetState(ctx context.Context, id int64) (*workflow.State, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.State), args.Error(1)
}
func (m *MockRepository) ListStates(ctx context.Context, wfID int64) ([]*workflow.State, error) {
	args := m.Called(ctx, wfID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*workflow.State), args.Error(1)
}
func (m *MockRepository) UpdateState(ctx context.Context, st *workflow.State) error {
	return m.Called(ctx, st).Error(0)
}
func (m *MockRepository) DeleteState(ctx context.Context, id int64) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockRepository) CreateTransition(ctx context.Context, tr *workflow.Transition) error {
	args := m.Called(ctx, tr)
	if tr.ID == 0 {
		tr.ID = 300
	}
	return args.Error(0)
}
func (m *MockRepository) DeleteTransition(ctx context.Context, id int64) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockRepository) GetNextState(ctx context.Context, curID int64, event string) (*workflow.State, error) {
	args := m.Called(ctx, curID, event)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.State), args.Error(1)
}

func (m *MockRepository) CreateStateAction(ctx context.Context, sa *workflow.StateAction) error {
	args := m.Called(ctx, sa)
	if sa.ID == 0 {
		sa.ID = 400
	}
	return args.Error(0)
}
func (m *MockRepository) ListStateActions(ctx context.Context, sID int64) ([]*workflow.StateAction, error) {
	args := m.Called(ctx, sID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*workflow.StateAction), args.Error(1)
}
func (m *MockRepository) DeleteStateAction(ctx context.Context, id int64) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockRepository) SetActiveWorkflow(ctx context.Context, id int64) (*workflow.Workflow, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.Workflow), args.Error(1)
}
func (m *MockRepository) GetActiveWorkflowByDepartment(ctx context.Context, depID int64) (*workflow.Workflow, error) {
	args := m.Called(ctx, depID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.Workflow), args.Error(1)
}

type MockWorkflowService struct {
	mock.Mock
}

func (m *MockWorkflowService) CreateWorkflow(ctx context.Context, name string, departmentID int64) (*workflow.Workflow, error) {
	args := m.Called(ctx, name, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.Workflow), args.Error(1)
}
func (m *MockWorkflowService) GetWorkflow(ctx context.Context, id int64) (*workflow.Workflow, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.Workflow), args.Error(1)
}
func (m *MockWorkflowService) GetWorkflowByName(ctx context.Context, name string) (*workflow.Workflow, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.Workflow), args.Error(1)
}
func (m *MockWorkflowService) ListWorkflows(ctx context.Context, departmentID int64) ([]*workflow.Workflow, error) {
	args := m.Called(ctx, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*workflow.Workflow), args.Error(1)
}
func (m *MockWorkflowService) UpdateWorkflow(ctx context.Context, id int64, name string) (*workflow.Workflow, error) {
	args := m.Called(ctx, id, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.Workflow), args.Error(1)
}
func (m *MockWorkflowService) DeleteWorkflow(ctx context.Context, id int64) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockWorkflowService) CreateState(ctx context.Context, req *workflowv1.CreateStateRequest) (*workflow.State, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.State), args.Error(1)
}
func (m *MockWorkflowService) GetState(ctx context.Context, id int64) (*workflow.State, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.State), args.Error(1)
}
func (m *MockWorkflowService) ListStates(ctx context.Context, workflowID int64) ([]*workflow.State, error) {
	args := m.Called(ctx, workflowID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*workflow.State), args.Error(1)
}
func (m *MockWorkflowService) UpdateState(ctx context.Context, req *workflowv1.UpdateStateRequest) (*workflow.State, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.State), args.Error(1)
}
func (m *MockWorkflowService) DeleteState(ctx context.Context, id int64) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockWorkflowService) CreateTransition(ctx context.Context, req *workflowv1.CreateTransitionRequest) (*workflow.Transition, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.Transition), args.Error(1)
}
func (m *MockWorkflowService) DeleteTransition(ctx context.Context, id int64) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockWorkflowService) CreateStateAction(ctx context.Context, req *workflowv1.CreateStateActionRequest) (*workflow.StateAction, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.StateAction), args.Error(1)
}
func (m *MockWorkflowService) ListStateActions(ctx context.Context, stateID int64) ([]*workflow.StateAction, error) {
	args := m.Called(ctx, stateID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*workflow.StateAction), args.Error(1)
}
func (m *MockWorkflowService) DeleteStateAction(ctx context.Context, id int64) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockWorkflowService) SetActiveWorkflow(ctx context.Context, workflowID int64) (*workflow.Workflow, error) {
	args := m.Called(ctx, workflowID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.Workflow), args.Error(1)
}
func (m *MockWorkflowService) GetActiveWorkflowByDepartment(ctx context.Context, departmentID int64) (*workflow.Workflow, error) {
	args := m.Called(ctx, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.Workflow), args.Error(1)
}
func (m *MockWorkflowService) GetNextState(ctx context.Context, currentStateID int64, eventName string) (*workflow.State, error) {
	args := m.Called(ctx, currentStateID, eventName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.State), args.Error(1)
}
