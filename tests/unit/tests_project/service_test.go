package tests_project

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lucky720s/diplomaflow/internal/project"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ================== Repository Mock ==================

type MockRepo struct {
	mock.Mock
}

func (m *MockRepo) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return errors.New("Transaction should not be called in these unit tests")
}

func (m *MockRepo) CreateWithOutbox(ctx context.Context, p *project.Project, eventType string, topic string, payloadBase map[string]interface{}) error {
	args := m.Called(ctx, p, eventType, topic, payloadBase)
	return args.Error(0)
}

func (m *MockRepo) GetByID(ctx context.Context, id int64) (*project.Project, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*project.Project), args.Error(1)
}

func (m *MockRepo) GetRuntimeByID(ctx context.Context, id int64) (*project.Project, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*project.Project), args.Error(1)
}

func (m *MockRepo) Update(ctx context.Context, p *project.Project) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *MockRepo) AddHistory(ctx context.Context, h *project.StateHistory) error {
	args := m.Called(ctx, h)
	return args.Error(0)
}

func (m *MockRepo) GetPendingEvents(ctx context.Context, limit int) ([]project.OutboxEvent, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]project.OutboxEvent), args.Error(1)
}

func (m *MockRepo) MarkEventProcessed(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepo) ListByStudent(ctx context.Context, studentID int64) ([]*project.Project, error) {
	args := m.Called(ctx, studentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*project.Project), args.Error(1)
}

func (m *MockRepo) GetProjectsWithExpiredDeadlines(ctx context.Context) ([]*project.Project, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*project.Project), args.Error(1)
}

// ================== Workflow Client Mock (narrow) ==================

type MockWorkflowClient struct {
	mock.Mock
}

func (m *MockWorkflowClient) GetActiveWorkflowByDepartment(ctx context.Context, in *workflowv1.GetActiveWorkflowByDepartmentRequest, _ ...interface{}) (*workflowv1.Workflow, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflowv1.Workflow), args.Error(1)
}

func (m *MockWorkflowClient) GetAvailableTransitions(ctx context.Context, in *workflowv1.GetAvailableTransitionsRequest, _ ...interface{}) (*workflowv1.GetAvailableTransitionsResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflowv1.GetAvailableTransitionsResponse), args.Error(1)
}

func (m *MockWorkflowClient) ExecuteTransition(ctx context.Context, in *workflowv1.ExecuteTransitionRequest, _ ...interface{}) (*workflowv1.ExecuteTransitionResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflowv1.ExecuteTransitionResponse), args.Error(1)
}

// ================== Tests ==================

func TestService_CreateProject_SetsInitialState(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepo)
	wf := new(MockWorkflowClient)
	log := zap.NewNop()

	svc := project.NewService(repo, wf, log)

	workflow := &workflowv1.Workflow{
		Id:      10,
		Name:    "WF",
		Version: 1,
		States: []*workflowv1.State{
			{Id: 100, Name: "TEAM_FORMATION", IsInitial: true, OrderIndex: 1, DurationDays: 14},
			{Id: 101, Name: "SUPERVISOR_SELECTION", IsInitial: false, OrderIndex: 2},
		},
	}

	wf.On("GetActiveWorkflowByDepartment", mock.Anything, &workflowv1.GetActiveWorkflowByDepartmentRequest{
		DepartmentId: 1,
	}).Return(workflow, nil).Once()

	repo.On(
		"CreateWithOutbox",
		mock.Anything,
		mock.AnythingOfType("*project.Project"),
		"ProjectCreated",
		"project-events",
		mock.MatchedBy(func(m map[string]interface{}) bool {
			return m["workflow_id"] == int64(10) &&
				m["initial_state_id"] == int64(100) &&
				m["initial_state"] == "TEAM_FORMATION"
		}),
	).Run(func(args mock.Arguments) {
		p := args.Get(1).(*project.Project)
		p.ID = 123
	}).Return(nil).Once()

	resp, err := svc.CreateProject(ctx, &projectv1.CreateProjectRequest{
		Title:        "T",
		Description:  "D",
		StudentId:    1000,
		WorkflowName: "ignored",
		UniversityId: 1,
		DepartmentId: 1,
	})
	require.NoError(t, err)
	require.Equal(t, int64(123), resp.ProjectId)
	require.Equal(t, "active", resp.Status)

	repo.AssertExpectations(t)
	wf.AssertExpectations(t)
}

func TestService_PerformAction_DelegatesToWorkflowTransition(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepo)
	wf := new(MockWorkflowClient)
	log := zap.NewNop()

	svc := project.NewService(repo, wf, log)

	repo.On("GetRuntimeByID", mock.Anything, int64(1)).Return(&project.Project{
		ID:               1,
		Status:           "active",
		CurrentStateID:   100,
		CurrentStateName: "TEAM_FORMATION",
		StudentID:        1000,
		DepartmentID:     1,
		UniversityID:     1,
	}, nil).Once()

	wf.On("GetAvailableTransitions", mock.Anything, &workflowv1.GetAvailableTransitionsRequest{
		ProjectId:      1,
		CurrentStateId: 100,
		UserId:         1000,
		UserRole:       "student",
	}).Return(&workflowv1.GetAvailableTransitionsResponse{
		Transitions: []*workflowv1.AvailableTransition{
			{
				Transition: &workflowv1.Transition{
					Id:        555,
					EventName: "TEAM_FORMED",
				},
				CanExecute: true,
			},
		},
	}, nil).Once()

	wf.On("ExecuteTransition", mock.Anything, mock.MatchedBy(func(r *workflowv1.ExecuteTransitionRequest) bool {
		return r.ProjectId == 1 && r.TransitionId == 555 && r.UserId == 1000
	})).Return(&workflowv1.ExecuteTransitionResponse{
		Success:      true,
		NewStateId:   101,
		NewStateName: "SUPERVISOR_SELECTION",
	}, nil).Once()

	repo.On("GetByID", mock.Anything, int64(1)).Return(&project.Project{
		ID:               1,
		Status:           "active",
		CurrentStateID:   101,
		CurrentStateName: "SUPERVISOR_SELECTION",
	}, nil).Once()

	p, err := svc.PerformAction(ctx, 1, "TEAM_FORMED", map[string]interface{}{}, 1000, "student")
	require.NoError(t, err)
	require.Equal(t, int64(101), p.CurrentStateID)
	require.Equal(t, "SUPERVISOR_SELECTION", p.CurrentStateName)

	repo.AssertExpectations(t)
	wf.AssertExpectations(t)
}

func TestService_PerformAction_BlockedTransition(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepo)
	wf := new(MockWorkflowClient)
	log := zap.NewNop()

	svc := project.NewService(repo, wf, log)

	repo.On("GetRuntimeByID", mock.Anything, int64(1)).Return(&project.Project{
		ID:             1,
		Status:         "active",
		CurrentStateID: 100,
	}, nil).Once()

	wf.On("GetAvailableTransitions", mock.Anything, &workflowv1.GetAvailableTransitionsRequest{
		ProjectId:      1,
		CurrentStateId: 100,
		UserId:         1000,
		UserRole:       "student",
	}).Return(&workflowv1.GetAvailableTransitionsResponse{
		Transitions: []*workflowv1.AvailableTransition{
			{
				Transition: &workflowv1.Transition{
					Id:        555,
					EventName: "TEAM_FORMED",
				},
				CanExecute:    false,
				BlockedReason: "missing requirements",
			},
		},
	}, nil).Once()

	_, err := svc.PerformAction(ctx, 1, "TEAM_FORMED", map[string]interface{}{}, 1000, "student")
	require.Error(t, err)

	repo.AssertExpectations(t)
	wf.AssertExpectations(t)
}

func TestService_GetProjectRuntime_MapsFields(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepo)
	wf := new(MockWorkflowClient)
	log := zap.NewNop()

	svc := project.NewService(repo, wf, log)

	teamID := int64(777)
	dl := time.Now().UTC().Add(24 * time.Hour)

	repo.On("GetRuntimeByID", mock.Anything, int64(1)).Return(&project.Project{
		ID:               1,
		StudentID:        1000,
		UniversityID:     1,
		DepartmentID:     2,
		TeamID:           &teamID,
		WorkflowID:       10,
		WorkflowVersion:  1,
		WorkflowName:     "WF",
		CurrentStateID:   100,
		CurrentStateName: "TEAM_FORMATION",
		Status:           "active",
		DeadlineAt:       &dl,
	}, nil).Once()

	resp, err := svc.GetProjectRuntime(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, int64(1), resp.ProjectId)
	require.Equal(t, int64(777), resp.TeamId)
	require.Equal(t, int64(100), resp.CurrentStateId)
	require.Equal(t, "TEAM_FORMATION", resp.CurrentStateName)
}
