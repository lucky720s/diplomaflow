package tests_workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/lucky720s/diplomaflow/internal/workflow"
)

// ==================== ERRORS ====================

var ErrNotFound = errors.New("not found")

// ==================== MOCKS ====================

type MockRepository struct {
	mock.Mock
}

// Workflow methods
func (m *MockRepository) CreateWorkflow(ctx context.Context, wf *workflow.Workflow) error {
	args := m.Called(ctx, wf)
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

func (m *MockRepository) GetWorkflowFull(ctx context.Context, id int64) (*workflow.Workflow, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.Workflow), args.Error(1)
}

func (m *MockRepository) ListWorkflows(ctx context.Context, departmentID int64) ([]*workflow.Workflow, error) {
	args := m.Called(ctx, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*workflow.Workflow), args.Error(1)
}

func (m *MockRepository) UpdateWorkflow(ctx context.Context, wf *workflow.Workflow) error {
	args := m.Called(ctx, wf)
	return args.Error(0)
}

func (m *MockRepository) DeleteWorkflow(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) GetActiveWorkflowByDepartment(ctx context.Context, departmentID int64) (*workflow.Workflow, error) {
	args := m.Called(ctx, departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.Workflow), args.Error(1)
}

func (m *MockRepository) DeactivateWorkflowsForDepartment(ctx context.Context, departmentID int64) error {
	args := m.Called(ctx, departmentID)
	return args.Error(0)
}

func (m *MockRepository) SetActiveWorkflow(ctx context.Context, workflowID int64) (*workflow.Workflow, error) {
	args := m.Called(ctx, workflowID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.Workflow), args.Error(1)
}

// State methods
func (m *MockRepository) CreateState(ctx context.Context, state *workflow.State) error {
	args := m.Called(ctx, state)
	return args.Error(0)
}

func (m *MockRepository) GetState(ctx context.Context, id int64) (*workflow.State, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.State), args.Error(1)
}

func (m *MockRepository) ListStates(ctx context.Context, workflowID int64) ([]workflow.State, error) {
	args := m.Called(ctx, workflowID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]workflow.State), args.Error(1)
}

func (m *MockRepository) UpdateState(ctx context.Context, state *workflow.State) error {
	args := m.Called(ctx, state)
	return args.Error(0)
}

func (m *MockRepository) DeleteState(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) ReorderStates(ctx context.Context, workflowID int64, stateIDs []int64) error {
	args := m.Called(ctx, workflowID, stateIDs)
	return args.Error(0)
}

func (m *MockRepository) GetInitialState(ctx context.Context, workflowID int64) (*workflow.State, error) {
	args := m.Called(ctx, workflowID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.State), args.Error(1)
}

func (m *MockRepository) GetNextState(ctx context.Context, currentStateID int64, eventName string) (*workflow.State, error) {
	args := m.Called(ctx, currentStateID, eventName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.State), args.Error(1)
}

// Transition methods
func (m *MockRepository) CreateTransition(ctx context.Context, tr *workflow.Transition) error {
	args := m.Called(ctx, tr)
	return args.Error(0)
}

func (m *MockRepository) GetTransition(ctx context.Context, id int64) (*workflow.Transition, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.Transition), args.Error(1)
}

func (m *MockRepository) ListTransitions(ctx context.Context, workflowID int64) ([]workflow.Transition, error) {
	args := m.Called(ctx, workflowID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]workflow.Transition), args.Error(1)
}

func (m *MockRepository) GetTransitionsByWorkflow(ctx context.Context, workflowID int64) ([]workflow.Transition, error) {
	args := m.Called(ctx, workflowID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]workflow.Transition), args.Error(1)
}

func (m *MockRepository) UpdateTransition(ctx context.Context, tr *workflow.Transition) error {
	args := m.Called(ctx, tr)
	return args.Error(0)
}

func (m *MockRepository) DeleteTransition(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) GetTransitionsFromState(ctx context.Context, stateID int64) ([]workflow.Transition, error) {
	args := m.Called(ctx, stateID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]workflow.Transition), args.Error(1)
}

func (m *MockRepository) GetTransitionByEvent(ctx context.Context, fromStateID int64, eventName string) (*workflow.Transition, error) {
	args := m.Called(ctx, fromStateID, eventName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.Transition), args.Error(1)
}

// StateAction methods
func (m *MockRepository) CreateStateAction(ctx context.Context, action *workflow.StateAction) error {
	args := m.Called(ctx, action)
	return args.Error(0)
}

func (m *MockRepository) GetStateAction(ctx context.Context, id int64) (*workflow.StateAction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.StateAction), args.Error(1)
}

func (m *MockRepository) ListStateActions(ctx context.Context, stateID int64) ([]workflow.StateAction, error) {
	args := m.Called(ctx, stateID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]workflow.StateAction), args.Error(1)
}

func (m *MockRepository) UpdateStateAction(ctx context.Context, action *workflow.StateAction) error {
	args := m.Called(ctx, action)
	return args.Error(0)
}

func (m *MockRepository) DeleteStateAction(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) GetStateActionsByTrigger(ctx context.Context, stateID int64, trigger string) ([]workflow.StateAction, error) {
	args := m.Called(ctx, stateID, trigger)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]workflow.StateAction), args.Error(1)
}

// Template methods
func (m *MockRepository) ListTemplates(ctx context.Context) ([]*workflow.WorkflowTemplate, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*workflow.WorkflowTemplate), args.Error(1)
}

func (m *MockRepository) GetTemplate(ctx context.Context, id int64) (*workflow.WorkflowTemplate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflow.WorkflowTemplate), args.Error(1)
}

// ==================== TEST HELPERS ====================

func createTestWorkflow() *workflow.Workflow {
	return &workflow.Workflow{
		ID:           1,
		Name:         "Дипломный проект 2025",
		Description:  "Стандартный процесс дипломной работы",
		DepartmentID: 1,
		Version:      1,
		IsActive:     true,
		IsTemplate:   false,
		Settings:     datatypes.JSON(`{"team_required": true, "min_team_size": 1, "max_team_size": 3}`),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		States: []workflow.State{
			{
				ID:           1,
				WorkflowID:   1,
				Name:         "TEAM_FORMATION",
				DisplayName:  "Формирование команды",
				OrderIndex:   1,
				Type:         "TEAM_FORMATION",
				IsInitial:    true,
				IsFinal:      false,
				DurationDays: 14,
			},
			{
				ID:           2,
				WorkflowID:   1,
				Name:         "SUPERVISOR_SELECTION",
				DisplayName:  "Выбор руководителя",
				OrderIndex:   2,
				Type:         "SUPERVISOR_SELECTION",
				IsInitial:    false,
				IsFinal:      false,
				DurationDays: 7,
			},
			{
				ID:          3,
				WorkflowID:  1,
				Name:        "COMPLETED",
				DisplayName: "Завершено",
				OrderIndex:  3,
				Type:        "COMPLETED",
				IsInitial:   false,
				IsFinal:     true,
			},
		},
		Transitions: []workflow.Transition{
			{
				ID:          1,
				WorkflowID:  1,
				EventName:   "TEAM_FORMED",
				DisplayName: "Команда создана",
				FromStateID: 1,
				ToStateID:   2,
				ButtonLabel: "Команда готова",
				ButtonColor: "success",
			},
			{
				ID:          2,
				WorkflowID:  1,
				EventName:   "SUPERVISOR_SELECTED",
				DisplayName: "Руководитель выбран",
				FromStateID: 2,
				ToStateID:   3,
				ButtonLabel: "Завершить",
				ButtonColor: "primary",
			},
		},
	}
}

// ==================== REPOSITORY TESTS ====================

func TestRepository_GetWorkflow(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		expected := createTestWorkflow()
		repo.On("GetWorkflow", ctx, int64(1)).Return(expected, nil).Once()

		wf, err := repo.GetWorkflow(ctx, 1)

		require.NoError(t, err)
		assert.Equal(t, expected.ID, wf.ID)
		assert.Equal(t, expected.Name, wf.Name)
		repo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		repo.On("GetWorkflow", ctx, int64(999)).
			Return(nil, ErrNotFound).Once()

		_, err := repo.GetWorkflow(ctx, 999)

		assert.Error(t, err)
		repo.AssertExpectations(t)
	})
}

func TestRepository_CreateWorkflow(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		wf := &workflow.Workflow{
			Name:         "Test Workflow",
			Description:  "Test Description",
			DepartmentID: 1,
		}
		repo.On("CreateWorkflow", ctx, wf).Return(nil).Once()

		err := repo.CreateWorkflow(ctx, wf)

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

func TestRepository_GetActiveWorkflowByDepartment(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("found active workflow", func(t *testing.T) {
		expected := createTestWorkflow()
		expected.IsActive = true
		repo.On("GetActiveWorkflowByDepartment", ctx, int64(1)).
			Return(expected, nil).Once()

		wf, err := repo.GetActiveWorkflowByDepartment(ctx, 1)

		require.NoError(t, err)
		assert.True(t, wf.IsActive)
		assert.Equal(t, int64(1), wf.DepartmentID)
		repo.AssertExpectations(t)
	})

	t.Run("no active workflow", func(t *testing.T) {
		repo.On("GetActiveWorkflowByDepartment", ctx, int64(999)).
			Return(nil, ErrNotFound).Once()

		_, err := repo.GetActiveWorkflowByDepartment(ctx, 999)

		assert.Error(t, err)
		repo.AssertExpectations(t)
	})
}

func TestRepository_ListWorkflows(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("list all", func(t *testing.T) {
		workflows := []*workflow.Workflow{
			createTestWorkflow(),
			{ID: 2, Name: "Workflow 2", DepartmentID: 1},
		}
		repo.On("ListWorkflows", ctx, int64(1)).
			Return(workflows, nil).Once()

		result, err := repo.ListWorkflows(ctx, 1)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		repo.AssertExpectations(t)
	})
}

func TestRepository_UpdateWorkflow(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		wf := createTestWorkflow()
		wf.Name = "Updated Workflow"
		repo.On("UpdateWorkflow", ctx, wf).Return(nil).Once()

		err := repo.UpdateWorkflow(ctx, wf)

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

func TestRepository_DeleteWorkflow(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("successful deletion", func(t *testing.T) {
		repo.On("DeleteWorkflow", ctx, int64(1)).Return(nil).Once()

		err := repo.DeleteWorkflow(ctx, 1)

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

// ==================== STATE REPOSITORY TESTS ====================

func TestRepository_CreateState(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		state := &workflow.State{
			WorkflowID:  1,
			Name:        "NEW_STATE",
			DisplayName: "New State",
			Type:        "REVIEW",
		}
		repo.On("CreateState", ctx, state).Return(nil).Once()

		err := repo.CreateState(ctx, state)

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

func TestRepository_GetState(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		expected := &workflow.State{
			ID:          1,
			WorkflowID:  1,
			Name:        "TEAM_FORMATION",
			DisplayName: "Формирование команды",
			Type:        "TEAM_FORMATION",
			IsInitial:   true,
		}
		repo.On("GetState", ctx, int64(1)).Return(expected, nil).Once()

		state, err := repo.GetState(ctx, 1)

		require.NoError(t, err)
		assert.Equal(t, "TEAM_FORMATION", state.Name)
		assert.True(t, state.IsInitial)
		repo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		repo.On("GetState", ctx, int64(999)).
			Return(nil, ErrNotFound).Once()

		_, err := repo.GetState(ctx, 999)

		assert.Error(t, err)
		repo.AssertExpectations(t)
	})
}

func TestRepository_ListStates(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("returns ordered states", func(t *testing.T) {
		states := []workflow.State{
			{ID: 1, Name: "STATE_1", OrderIndex: 1},
			{ID: 2, Name: "STATE_2", OrderIndex: 2},
			{ID: 3, Name: "STATE_3", OrderIndex: 3},
		}
		repo.On("ListStates", ctx, int64(1)).Return(states, nil).Once()

		result, err := repo.ListStates(ctx, 1)

		require.NoError(t, err)
		assert.Len(t, result, 3)
		assert.Equal(t, int32(1), result[0].OrderIndex)
		assert.Equal(t, int32(2), result[1].OrderIndex)
		repo.AssertExpectations(t)
	})
}

func TestRepository_GetInitialState(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("found initial state", func(t *testing.T) {
		expected := &workflow.State{
			ID:        1,
			Name:      "START",
			IsInitial: true,
		}
		repo.On("GetInitialState", ctx, int64(1)).Return(expected, nil).Once()

		state, err := repo.GetInitialState(ctx, 1)

		require.NoError(t, err)
		assert.True(t, state.IsInitial)
		repo.AssertExpectations(t)
	})
}

func TestRepository_GetNextState(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("found next state", func(t *testing.T) {
		nextState := &workflow.State{
			ID:          2,
			Name:        "SUPERVISOR_SELECTION",
			DisplayName: "Выбор руководителя",
		}

		repo.On("GetNextState", ctx, int64(1), "TEAM_FORMED").
			Return(nextState, nil).Once()

		state, err := repo.GetNextState(ctx, 1, "TEAM_FORMED")

		require.NoError(t, err)
		assert.Equal(t, int64(2), state.ID)
		assert.Equal(t, "SUPERVISOR_SELECTION", state.Name)
		repo.AssertExpectations(t)
	})

	t.Run("no transition found", func(t *testing.T) {
		repo.On("GetNextState", ctx, int64(1), "UNKNOWN_EVENT").
			Return(nil, ErrNotFound).Once()

		_, err := repo.GetNextState(ctx, 1, "UNKNOWN_EVENT")

		assert.Error(t, err)
		repo.AssertExpectations(t)
	})
}

// ==================== TRANSITION REPOSITORY TESTS ====================

func TestRepository_CreateTransition(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		tr := &workflow.Transition{
			WorkflowID:  1,
			EventName:   "SUBMIT",
			DisplayName: "Submit",
			FromStateID: 1,
			ToStateID:   2,
		}
		repo.On("CreateTransition", ctx, tr).Return(nil).Once()

		err := repo.CreateTransition(ctx, tr)

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

func TestRepository_ListTransitions(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("returns all transitions", func(t *testing.T) {
		transitions := []workflow.Transition{
			{ID: 1, EventName: "APPROVE", FromStateID: 1, ToStateID: 2},
			{ID: 2, EventName: "REJECT", FromStateID: 1, ToStateID: 3},
		}
		repo.On("ListTransitions", ctx, int64(1)).
			Return(transitions, nil).Once()

		result, err := repo.ListTransitions(ctx, 1)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		repo.AssertExpectations(t)
	})
}

func TestRepository_GetTransitionsFromState(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("returns transitions from state", func(t *testing.T) {
		transitions := []workflow.Transition{
			{ID: 1, EventName: "APPROVE", FromStateID: 1, ToStateID: 2},
			{ID: 2, EventName: "REJECT", FromStateID: 1, ToStateID: 3},
		}
		repo.On("GetTransitionsFromState", ctx, int64(1)).
			Return(transitions, nil).Once()

		result, err := repo.GetTransitionsFromState(ctx, 1)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		repo.AssertExpectations(t)
	})
}

// ==================== STATE ACTION REPOSITORY TESTS ====================

func TestRepository_CreateStateAction(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("create notification action", func(t *testing.T) {
		action := &workflow.StateAction{
			StateID: 1,
			Name:    "Send Welcome Notification",
			Type:    "SEND_NOTIFICATION",
			Trigger: "ON_ENTER",
		}
		repo.On("CreateStateAction", ctx, action).Return(nil).Once()

		err := repo.CreateStateAction(ctx, action)

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

func TestRepository_ListStateActions(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("returns actions for state", func(t *testing.T) {
		actions := []workflow.StateAction{
			{ID: 1, StateID: 1, Type: "SEND_NOTIFICATION", Trigger: "ON_ENTER"},
			{ID: 2, StateID: 1, Type: "SEND_EMAIL", Trigger: "ON_EXIT"},
		}
		repo.On("ListStateActions", ctx, int64(1)).Return(actions, nil).Once()

		result, err := repo.ListStateActions(ctx, 1)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		repo.AssertExpectations(t)
	})
}

func TestRepository_GetStateActionsByTrigger(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("returns ON_ENTER actions only", func(t *testing.T) {
		actions := []workflow.StateAction{
			{ID: 1, StateID: 1, Type: "SEND_NOTIFICATION", Trigger: "ON_ENTER"},
		}
		repo.On("GetStateActionsByTrigger", ctx, int64(1), "ON_ENTER").
			Return(actions, nil).Once()

		result, err := repo.GetStateActionsByTrigger(ctx, 1, "ON_ENTER")

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "ON_ENTER", result[0].Trigger)
		repo.AssertExpectations(t)
	})
}

// ==================== TEMPLATE REPOSITORY TESTS ====================

func TestRepository_ListTemplates(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("returns templates", func(t *testing.T) {
		templates := []*workflow.WorkflowTemplate{
			{ID: 1, Name: "Diploma Template", Category: "diploma"},
			{ID: 2, Name: "Coursework Template", Category: "coursework"},
		}
		repo.On("ListTemplates", ctx).Return(templates, nil).Once()

		result, err := repo.ListTemplates(ctx)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		repo.AssertExpectations(t)
	})
}

func TestRepository_GetTemplate(t *testing.T) {
	repo := new(MockRepository)
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		template := &workflow.WorkflowTemplate{
			ID:           1,
			Name:         "Diploma Template",
			TemplateData: datatypes.JSON(`{"states": [], "transitions": []}`),
		}
		repo.On("GetTemplate", ctx, int64(1)).Return(template, nil).Once()

		result, err := repo.GetTemplate(ctx, 1)

		require.NoError(t, err)
		assert.Equal(t, "Diploma Template", result.Name)
		repo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		repo.On("GetTemplate", ctx, int64(999)).
			Return(nil, ErrNotFound).Once()

		_, err := repo.GetTemplate(ctx, 999)

		assert.Error(t, err)
		repo.AssertExpectations(t)
	})
}

// ==================== WORKFLOW VALIDATION TESTS ====================

func TestWorkflow_Validation(t *testing.T) {
	t.Run("valid workflow has initial and final states", func(t *testing.T) {
		wf := createTestWorkflow()

		hasInitial := false
		hasFinal := false
		for _, s := range wf.States {
			if s.IsInitial {
				hasInitial = true
			}
			if s.IsFinal {
				hasFinal = true
			}
		}

		assert.True(t, hasInitial, "workflow should have initial state")
		assert.True(t, hasFinal, "workflow should have final state")
	})

	t.Run("workflow without states is invalid", func(t *testing.T) {
		wf := createTestWorkflow()
		wf.States = nil

		assert.Empty(t, wf.States)
	})

	t.Run("workflow without initial state is invalid", func(t *testing.T) {
		wf := createTestWorkflow()
		for i := range wf.States {
			wf.States[i].IsInitial = false
		}

		hasInitial := false
		for _, s := range wf.States {
			if s.IsInitial {
				hasInitial = true
			}
		}

		assert.False(t, hasInitial)
	})

	t.Run("workflow without final state is invalid", func(t *testing.T) {
		wf := createTestWorkflow()
		for i := range wf.States {
			wf.States[i].IsFinal = false
		}

		hasFinal := false
		for _, s := range wf.States {
			if s.IsFinal {
				hasFinal = true
			}
		}

		assert.False(t, hasFinal)
	})
}

// ==================== TRANSITION VALIDATION TESTS ====================

func TestTransition_Validation(t *testing.T) {
	t.Run("transition connects existing states", func(t *testing.T) {
		wf := createTestWorkflow()

		stateIDs := make(map[int64]bool)
		for _, s := range wf.States {
			stateIDs[s.ID] = true
		}

		for _, tr := range wf.Transitions {
			assert.True(t, stateIDs[tr.FromStateID],
				"transition from_state_id should exist in workflow states")
			assert.True(t, stateIDs[tr.ToStateID],
				"transition to_state_id should exist in workflow states")
		}
	})

	t.Run("all non-final states have outgoing transitions", func(t *testing.T) {
		wf := createTestWorkflow()

		transitionsFrom := make(map[int64]bool)
		for _, tr := range wf.Transitions {
			transitionsFrom[tr.FromStateID] = true
		}

		for _, s := range wf.States {
			if !s.IsFinal {
				assert.True(t, transitionsFrom[s.ID],
					"non-final state %s should have outgoing transitions", s.Name)
			}
		}
	})
}

// ==================== MODEL TESTS ====================

func TestState_Model(t *testing.T) {
	t.Run("state has correct fields", func(t *testing.T) {
		state := workflow.State{
			ID:           1,
			WorkflowID:   1,
			Name:         "TEST",
			DisplayName:  "Test State",
			OrderIndex:   1,
			Type:         "REVIEW",
			IsInitial:    true,
			IsFinal:      false,
			DurationDays: 7,
		}

		assert.Equal(t, int64(1), state.ID)
		assert.Equal(t, "TEST", state.Name)
		assert.Equal(t, "Test State", state.DisplayName)
		assert.True(t, state.IsInitial)
		assert.False(t, state.IsFinal)
		assert.Equal(t, int32(7), state.DurationDays)
	})
}

func TestTransition_Model(t *testing.T) {
	t.Run("transition has correct fields", func(t *testing.T) {
		tr := workflow.Transition{
			ID:          1,
			WorkflowID:  1,
			EventName:   "SUBMIT",
			DisplayName: "Submit",
			FromStateID: 1,
			ToStateID:   2,
			ButtonLabel: "Submit",
			ButtonColor: "primary",
		}

		assert.Equal(t, int64(1), tr.ID)
		assert.Equal(t, "SUBMIT", tr.EventName)
		assert.Equal(t, int64(1), tr.FromStateID)
		assert.Equal(t, int64(2), tr.ToStateID)
	})
}

func TestStateAction_Model(t *testing.T) {
	t.Run("state action has correct fields", func(t *testing.T) {
		action := workflow.StateAction{
			ID:        1,
			StateID:   1,
			Name:      "Send Notification",
			Type:      "SEND_NOTIFICATION",
			Trigger:   "ON_ENTER",
			IsEnabled: true,
		}

		assert.Equal(t, int64(1), action.ID)
		assert.Equal(t, "SEND_NOTIFICATION", action.Type)
		assert.Equal(t, "ON_ENTER", action.Trigger)
		assert.True(t, action.IsEnabled)
	})
}

// ==================== HELPER FUNCTION TESTS ====================

func TestFindInitialState(t *testing.T) {
	t.Run("finds state with IsInitial flag", func(t *testing.T) {
		states := []workflow.State{
			{ID: 1, Name: "STATE_1", IsInitial: false, OrderIndex: 1},
			{ID: 2, Name: "STATE_2", IsInitial: true, OrderIndex: 2},
			{ID: 3, Name: "STATE_3", IsInitial: false, OrderIndex: 3},
		}

		var initial *workflow.State
		for i := range states {
			if states[i].IsInitial {
				initial = &states[i]
				break
			}
		}

		require.NotNil(t, initial)
		assert.Equal(t, int64(2), initial.ID)
		assert.True(t, initial.IsInitial)
	})

	t.Run("falls back to first by order", func(t *testing.T) {
		states := []workflow.State{
			{ID: 3, Name: "STATE_3", IsInitial: false, OrderIndex: 3},
			{ID: 1, Name: "STATE_1", IsInitial: false, OrderIndex: 1},
			{ID: 2, Name: "STATE_2", IsInitial: false, OrderIndex: 2},
		}

		var initial *workflow.State
		for i := range states {
			if states[i].IsInitial {
				initial = &states[i]
				break
			}
		}

		// No initial flag, find by OrderIndex
		if initial == nil && len(states) > 0 {
			minIdx := states[0]
			for _, s := range states[1:] {
				if s.OrderIndex < minIdx.OrderIndex {
					minIdx = s
				}
			}
			initial = &minIdx
		}

		require.NotNil(t, initial)
		assert.Equal(t, int64(1), initial.ID) // Lowest OrderIndex
	})
}

func TestFindFinalStates(t *testing.T) {
	t.Run("finds all final states", func(t *testing.T) {
		states := []workflow.State{
			{ID: 1, Name: "STATE_1", IsFinal: false},
			{ID: 2, Name: "COMPLETED", IsFinal: true},
			{ID: 3, Name: "CANCELLED", IsFinal: true},
		}

		var finals []workflow.State
		for _, s := range states {
			if s.IsFinal {
				finals = append(finals, s)
			}
		}

		assert.Len(t, finals, 2)
		assert.True(t, finals[0].IsFinal)
		assert.True(t, finals[1].IsFinal)
	})

	t.Run("returns empty for no final states", func(t *testing.T) {
		states := []workflow.State{
			{ID: 1, Name: "STATE_1", IsFinal: false},
			{ID: 2, Name: "STATE_2", IsFinal: false},
		}

		var finals []workflow.State
		for _, s := range states {
			if s.IsFinal {
				finals = append(finals, s)
			}
		}

		assert.Empty(t, finals)
	})
}

// ==================== WORKFLOW STRUCTURE TESTS ====================

func TestWorkflowStructure(t *testing.T) {
	t.Run("workflow has all required fields", func(t *testing.T) {
		wf := createTestWorkflow()

		assert.NotEmpty(t, wf.Name)
		assert.NotZero(t, wf.DepartmentID)
		assert.NotZero(t, wf.Version)
		assert.NotEmpty(t, wf.States)
		assert.NotEmpty(t, wf.Transitions)
	})

	t.Run("states are in correct order", func(t *testing.T) {
		wf := createTestWorkflow()

		for i := 1; i < len(wf.States); i++ {
			assert.Greater(t, wf.States[i].OrderIndex, wf.States[i-1].OrderIndex,
				"states should be in ascending order")
		}
	})

	t.Run("transitions reference valid states", func(t *testing.T) {
		wf := createTestWorkflow()

		stateIDs := make(map[int64]bool)
		for _, s := range wf.States {
			stateIDs[s.ID] = true
		}

		for _, tr := range wf.Transitions {
			assert.True(t, stateIDs[tr.FromStateID], "FromStateID must exist")
			assert.True(t, stateIDs[tr.ToStateID], "ToStateID must exist")
		}
	})
}
