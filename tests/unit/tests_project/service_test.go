package tests_project

import (
	"context"
	"testing"
	"time"

	"github.com/lucky720s/diplomaflow/internal/project"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// ==================== MOCKS ====================

type MockRepo struct{ mock.Mock }

func (m *MockRepo) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return m.Called(ctx, mock.Anything).Error(0)
}
func (m *MockRepo) CreateWithOutbox(ctx context.Context, p *project.Project, eventType string, topic string, payloadBase map[string]interface{}) error {
	return m.Called(ctx, p, eventType, topic, payloadBase).Error(0)
}
func (m *MockRepo) GetByID(ctx context.Context, id int64) (*project.Project, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*project.Project), args.Error(1)
}
func (m *MockRepo) GetRuntimeByID(ctx context.Context, id int64) (*project.Project, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*project.Project), args.Error(1)
}
func (m *MockRepo) GetPendingEvents(ctx context.Context, limit int) ([]project.OutboxEvent, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]project.OutboxEvent), args.Error(1)
}
func (m *MockRepo) MarkEventProcessed(ctx context.Context, id int64) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockRepo) ListByStudent(ctx context.Context, studentID int64) ([]*project.Project, error) {
	args := m.Called(ctx, studentID)
	return args.Get(0).([]*project.Project), args.Error(1)
}
func (m *MockRepo) GetProjectsWithExpiredDeadlines(ctx context.Context) ([]*project.Project, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*project.Project), args.Error(1)
}
func (m *MockRepo) ListProjectsRuntime(ctx context.Context, f project.ProjectFilter) ([]*project.Project, int64, error) {
	args := m.Called(ctx, f)
	return args.Get(0).([]*project.Project), args.Get(1).(int64), args.Error(2)
}
func (m *MockRepo) ListStateHistory(ctx context.Context, projectID int64, limit int, order string) ([]project.StateHistory, error) {
	args := m.Called(ctx, projectID, limit, order)
	return args.Get(0).([]project.StateHistory), args.Error(1)
}

type MockWorkflowClient struct{ mock.Mock }

func (m *MockWorkflowClient) GetActiveWorkflowByDepartment(ctx context.Context, in *workflowv1.GetActiveWorkflowByDepartmentRequest, _ ...grpc.CallOption) (*workflowv1.Workflow, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*workflowv1.Workflow), args.Error(1)
}
func (m *MockWorkflowClient) GetAvailableTransitions(ctx context.Context, in *workflowv1.GetAvailableTransitionsRequest, _ ...grpc.CallOption) (*workflowv1.GetAvailableTransitionsResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*workflowv1.GetAvailableTransitionsResponse), args.Error(1)
}
func (m *MockWorkflowClient) ExecuteTransition(ctx context.Context, in *workflowv1.ExecuteTransitionRequest, _ ...grpc.CallOption) (*workflowv1.ExecuteTransitionResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*workflowv1.ExecuteTransitionResponse), args.Error(1)
}

// ==================== CREATE PROJECT TESTS ====================

func TestCreateProject_SetsInitialState(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepo)
	wf := new(MockWorkflowClient)
	log := zap.NewNop()

	svc := project.NewService(repo, wf, log)

	wf.On("GetActiveWorkflowByDepartment", mock.Anything, &workflowv1.GetActiveWorkflowByDepartmentRequest{DepartmentId: 1}).
		Return(&workflowv1.Workflow{
			Id:      10,
			Name:    "WF",
			Version: 1,
			States: []*workflowv1.State{
				{Id: 100, Name: "TEAM_FORMATION", IsInitial: true, DurationDays: 7},
				{Id: 101, Name: "NEXT", IsInitial: false},
			},
		}, nil).Once()

	repo.On("CreateWithOutbox", mock.Anything, mock.AnythingOfType("*project.Project"), "ProjectCreated", "project-events", mock.Anything).
		Run(func(args mock.Arguments) {
			p := args.Get(1).(*project.Project)
			p.ID = 123
		}).
		Return(nil).Once()

	resp, err := svc.CreateProject(ctx, &projectv1.CreateProjectRequest{
		Title:        "t",
		Description:  "d",
		StudentId:    1,
		UniversityId: 1,
		DepartmentId: 1,
	})
	require.NoError(t, err)
	require.Equal(t, int64(123), resp.ProjectId)
	require.Equal(t, "TEAM_FORMATION", resp.Status)
}

func TestCreateProject_SetsDeadlineFromDurationDays(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepo)
	wf := new(MockWorkflowClient)
	svc := project.NewService(repo, wf, zap.NewNop())

	wf.On("GetActiveWorkflowByDepartment", mock.Anything, mock.Anything).
		Return(&workflowv1.Workflow{
			Id:   10,
			Name: "WF",
			States: []*workflowv1.State{
				{Id: 1, Name: "START", IsInitial: true, DurationDays: 14},
			},
		}, nil).Once()

	var capturedProject *project.Project
	repo.On("CreateWithOutbox", mock.Anything, mock.AnythingOfType("*project.Project"), mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			capturedProject = args.Get(1).(*project.Project)
			capturedProject.ID = 1
		}).Return(nil).Once()

	_, err := svc.CreateProject(ctx, &projectv1.CreateProjectRequest{
		Title: "test", StudentId: 1, UniversityId: 1, DepartmentId: 1,
	})
	require.NoError(t, err)
	require.NotNil(t, capturedProject.DeadlineAt, "deadline should be set from DurationDays")
	expectedDeadline := time.Now().UTC().AddDate(0, 0, 14)
	delta := capturedProject.DeadlineAt.Sub(expectedDeadline)
	if delta < 0 {
		delta = -delta
	}
	require.Less(t, delta, 5*time.Second, "deadline should be ~14 days from now")
}

func TestCreateProject_FailsIfNoActiveWorkflow(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepo)
	wf := new(MockWorkflowClient)
	svc := project.NewService(repo, wf, zap.NewNop())

	wf.On("GetActiveWorkflowByDepartment", mock.Anything, mock.Anything).
		Return((*workflowv1.Workflow)(nil), status.Error(codes.NotFound, "no active workflow")).Once()

	_, err := svc.CreateProject(ctx, &projectv1.CreateProjectRequest{
		Title: "t", StudentId: 1, UniversityId: 1, DepartmentId: 99,
	})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "error should be a gRPC status error")
	require.Equal(t, codes.FailedPrecondition, st.Code(), "should return FailedPrecondition when no active workflow")
}

func TestCreateProject_SetsWorkflowFields(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepo)
	wf := new(MockWorkflowClient)
	svc := project.NewService(repo, wf, zap.NewNop())

	wf.On("GetActiveWorkflowByDepartment", mock.Anything, mock.Anything).
		Return(&workflowv1.Workflow{
			Id: 42, Name: "DipWF", Version: 3,
			States: []*workflowv1.State{
				{Id: 7, Name: "INIT_STATE", IsInitial: true},
			},
		}, nil).Once()

	var capturedProject *project.Project
	repo.On("CreateWithOutbox", mock.Anything, mock.AnythingOfType("*project.Project"), mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			capturedProject = args.Get(1).(*project.Project)
			capturedProject.ID = 5
		}).Return(nil).Once()

	_, err := svc.CreateProject(ctx, &projectv1.CreateProjectRequest{
		Title: "t", StudentId: 1, UniversityId: 1, DepartmentId: 1,
	})
	require.NoError(t, err)
	require.Equal(t, int64(42), capturedProject.WorkflowID)
	require.Equal(t, "DipWF", capturedProject.WorkflowName)
	require.Equal(t, int32(3), capturedProject.WorkflowVersion)
	require.Equal(t, int64(7), capturedProject.CurrentStateID)
	require.Equal(t, "INIT_STATE", capturedProject.CurrentStateName)
}

// ==================== PERFORM ACTION TESTS ====================

func TestPerformAction_DelegatesToWorkflow(t *testing.T) {
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
	}, nil).Once()

	wf.On("GetAvailableTransitions", mock.Anything, &workflowv1.GetAvailableTransitionsRequest{
		ProjectId:      1,
		CurrentStateId: 100,
		UserId:         10,
		UserRole:       "student",
	}).Return(&workflowv1.GetAvailableTransitionsResponse{
		Transitions: []*workflowv1.AvailableTransition{
			{Transition: &workflowv1.Transition{Id: 555, EventName: "TEAM_FORMED"}, CanExecute: true},
		},
	}, nil).Once()

	wf.On(
		"ExecuteTransition",
		mock.Anything,
		mock.MatchedBy(func(r *workflowv1.ExecuteTransitionRequest) bool {
			return r.GetProjectId() == 1 &&
				r.GetTransitionId() == 555 &&
				r.GetUserId() == 10
		}),
	).Return(
		&workflowv1.ExecuteTransitionResponse{Success: true, NewStateId: 101, NewStateName: "NEXT"},
		nil,
	).Once()

	repo.On("GetByID", mock.Anything, int64(1)).Return(&project.Project{
		ID:               1,
		Status:           "active",
		CurrentStateID:   101,
		CurrentStateName: "NEXT",
	}, nil).Once()

	p, err := svc.PerformAction(ctx, 1, "TEAM_FORMED", map[string]interface{}{}, 10, "student")
	require.NoError(t, err)
	require.Equal(t, int64(101), p.CurrentStateID)
	require.Equal(t, "NEXT", p.CurrentStateName)
}

func TestPerformAction_BlockedTransitionReturnsError(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepo)
	wf := new(MockWorkflowClient)
	svc := project.NewService(repo, wf, zap.NewNop())

	repo.On("GetRuntimeByID", mock.Anything, int64(1)).Return(&project.Project{
		ID: 1, Status: "active", CurrentStateID: 100,
	}, nil).Once()

	wf.On("GetAvailableTransitions", mock.Anything, mock.Anything).
		Return(&workflowv1.GetAvailableTransitionsResponse{
			Transitions: []*workflowv1.AvailableTransition{
				{Transition: &workflowv1.Transition{Id: 1, EventName: "SUBMIT"}, CanExecute: false, BlockedReason: "team not formed"},
			},
		}, nil).Once()

	_, err := svc.PerformAction(ctx, 1, "SUBMIT", map[string]interface{}{}, 10, "student")
	require.Error(t, err)
	require.Contains(t, err.Error(), "transition blocked")
}

func TestPerformAction_UnknownActionReturnsError(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepo)
	wf := new(MockWorkflowClient)
	svc := project.NewService(repo, wf, zap.NewNop())

	repo.On("GetRuntimeByID", mock.Anything, int64(1)).Return(&project.Project{
		ID: 1, Status: "active", CurrentStateID: 100,
	}, nil).Once()

	wf.On("GetAvailableTransitions", mock.Anything, mock.Anything).
		Return(&workflowv1.GetAvailableTransitionsResponse{
			Transitions: []*workflowv1.AvailableTransition{
				{Transition: &workflowv1.Transition{Id: 1, EventName: "OTHER_ACTION"}, CanExecute: true},
			},
		}, nil).Once()

	_, err := svc.PerformAction(ctx, 1, "UNKNOWN_ACTION", map[string]interface{}{}, 10, "student")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown action/transition")
}

func TestPerformAction_SkipsCompletedProject(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepo)
	wf := new(MockWorkflowClient)
	svc := project.NewService(repo, wf, zap.NewNop())

	repo.On("GetRuntimeByID", mock.Anything, int64(1)).Return(&project.Project{
		ID: 1, Status: "completed", CurrentStateID: 100,
	}, nil).Once()

	_, err := svc.PerformAction(ctx, 1, "SOME_ACTION", map[string]interface{}{}, 10, "student")
	require.Error(t, err)
	require.Contains(t, err.Error(), "inactive project")
	wf.AssertNotCalled(t, "GetAvailableTransitions")
}

// ==================== GET PROJECT RUNTIME TESTS ====================

func TestGetProjectRuntime_MapsFields(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepo)
	wf := new(MockWorkflowClient)
	log := zap.NewNop()
	svc := project.NewService(repo, wf, log)

	teamID := int64(77)
	dl := time.Now().UTC().Add(24 * time.Hour)

	repo.On("GetRuntimeByID", mock.Anything, int64(1)).Return(&project.Project{
		ID:               1,
		StudentID:        2,
		UniversityID:     3,
		DepartmentID:     4,
		TeamID:           teamID,
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
	require.Equal(t, int64(77), resp.TeamId)
	require.Equal(t, int64(100), resp.CurrentStateId)
	require.Equal(t, int64(10), resp.WorkflowId)
	require.Equal(t, "WF", resp.WorkflowName)
	require.Equal(t, "TEAM_FORMATION", resp.CurrentStateName)
	require.NotNil(t, resp.DeadlineAt)
}

func TestGetProjectRuntime_NilDeadlineOmitted(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepo)
	wf := new(MockWorkflowClient)
	svc := project.NewService(repo, wf, zap.NewNop())

	repo.On("GetRuntimeByID", mock.Anything, int64(1)).Return(&project.Project{
		ID: 1, WorkflowID: 10, CurrentStateID: 100, Status: "active",
	}, nil).Once()

	resp, err := svc.GetProjectRuntime(ctx, 1)
	require.NoError(t, err)
	require.Nil(t, resp.DeadlineAt)
}

// ==================== LIST PROJECTS RUNTIME TESTS ====================

func TestListProjectsRuntime_FiltersByDepartment(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepo)
	wf := new(MockWorkflowClient)
	svc := project.NewService(repo, wf, zap.NewNop())

	expectedFilter := project.ProjectFilter{
		DepartmentID: 5,
		Page:         1,
		PageSize:     20,
	}

	repo.On("ListProjectsRuntime", mock.Anything, expectedFilter).
		Return([]*project.Project{
			{ID: 1, DepartmentID: 5, WorkflowID: 10, Status: "active"},
			{ID: 2, DepartmentID: 5, WorkflowID: 10, Status: "active"},
		}, int64(2), nil).Once()

	resp, err := svc.ListProjectsRuntime(ctx, &projectv1.ListProjectsRuntimeRequest{
		DepartmentId: 5,
		Page:         1,
		PageSize:     20,
	})
	require.NoError(t, err)
	require.Equal(t, int32(2), resp.Total)
	require.Len(t, resp.Projects, 2)
}

func TestListProjectsRuntime_EmptyResult(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepo)
	wf := new(MockWorkflowClient)
	svc := project.NewService(repo, wf, zap.NewNop())

	repo.On("ListProjectsRuntime", mock.Anything, mock.Anything).
		Return([]*project.Project{}, int64(0), nil).Once()

	resp, err := svc.ListProjectsRuntime(ctx, &projectv1.ListProjectsRuntimeRequest{DepartmentId: 1})
	require.NoError(t, err)
	require.Equal(t, int32(0), resp.Total)
	require.Empty(t, resp.Projects)
}

// ==================== LIST STATE HISTORY TESTS ====================

func TestListProjectStateHistory_ReturnsHistoryItems(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepo)
	wf := new(MockWorkflowClient)
	svc := project.NewService(repo, wf, zap.NewNop())

	fromID := int64(100)
	toID := int64(101)
	changedBy := int64(42)
	trID := int64(7)
	repo.On("ListStateHistory", mock.Anything, int64(1), 10, "desc").
		Return([]project.StateHistory{
			{
				ID: 1, ProjectID: 1, EventName: "TEAM_FORMED",
				FromStateID: &fromID, ToStateID: &toID,
				FromStateName: "TEAM_FORMATION", ToStateName: "SUPERVISOR_SELECTION",
				ChangedBy: &changedBy, Direction: "forward", ActorRole: "admin",
				TransitionID: &trID,
			},
		}, nil).Once()

	resp, err := svc.ListProjectStateHistory(ctx, &projectv1.ListProjectStateHistoryRequest{
		ProjectId: 1, Limit: 10, Order: "desc",
	})
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	item := resp.Items[0]
	require.Equal(t, "TEAM_FORMED", item.EventName)
	require.Equal(t, int64(100), item.FromStateId)
	require.Equal(t, int64(101), item.ToStateId)
	require.Equal(t, "forward", item.Direction)
	require.Equal(t, "admin", item.ActorRole)
	require.Equal(t, int64(7), item.TransitionId)
	require.Equal(t, int64(42), item.ChangedBy)
}

func TestListProjectStateHistory_EmptyResult(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepo)
	wf := new(MockWorkflowClient)
	svc := project.NewService(repo, wf, zap.NewNop())

	repo.On("ListStateHistory", mock.Anything, int64(99), 50, "desc").
		Return([]project.StateHistory{}, nil).Once()

	resp, err := svc.ListProjectStateHistory(ctx, &projectv1.ListProjectStateHistoryRequest{
		ProjectId: 99, Limit: 50, Order: "desc",
	})
	require.NoError(t, err)
	require.Empty(t, resp.Items)
}
