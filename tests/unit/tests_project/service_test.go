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
	"gorm.io/gorm"
)

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

type MockWorkflowClient struct{ mock.Mock }

// ВАЖНО: opts ...grpc.CallOption (как в project.WorkflowClient)
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
	require.Equal(t, "active", resp.Status)
}

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
		mock.Anything, // ctx
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
	require.Equal(t, int64(77), resp.TeamId)
	require.Equal(t, int64(100), resp.CurrentStateId)
}
