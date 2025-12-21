package tests_project

import (
	"context"

	"github.com/lucky720s/diplomaflow/internal/project"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// --- Mock Repository ---
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateWithOutbox(ctx context.Context, p *project.Project, et, t string, pl map[string]interface{}) error {
	args := m.Called(ctx, p, et, t, pl)
	if p.ID == 0 {
		p.ID = 100
	}
	return args.Error(0)
}
func (m *MockRepository) GetByID(ctx context.Context, id uint64) (*project.Project, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*project.Project), args.Error(1)
}
func (m *MockRepository) Update(ctx context.Context, p *project.Project) error {
	return m.Called(ctx, p).Error(0)
}
func (m *MockRepository) ListByStudent(ctx context.Context, sid int64) ([]*project.Project, error) {
	args := m.Called(ctx, sid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*project.Project), args.Error(1)
}
func (m *MockRepository) AddHistory(ctx context.Context, h *project.StateHistory) error {
	return m.Called(ctx, h).Error(0)
}
func (m *MockRepository) ListAll(ctx context.Context, did int64, l, o int) ([]*project.Project, int64, error) {
	args := m.Called(ctx, did, l, o)
	return args.Get(0).([]*project.Project), args.Get(1).(int64), args.Error(2)
}
func (m *MockRepository) GetPendingEvents(ctx context.Context, limit int) ([]project.OutboxEvent, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]project.OutboxEvent), args.Error(1)
}
func (m *MockRepository) DeleteEvent(ctx context.Context, id uint) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockRepository) MarkEventProcessed(ctx context.Context, id uint) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockRepository) GetProjectsWithExpiredDeadlines(ctx context.Context) ([]*project.Project, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*project.Project), args.Error(1)
}

// --- Mock Service (для Handler) ---
type MockProjectService struct {
	mock.Mock
}

func (m *MockProjectService) CreateProject(ctx context.Context, req *projectv1.CreateProjectRequest) (*projectv1.CreateProjectResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*projectv1.CreateProjectResponse), args.Error(1)
}
func (m *MockProjectService) GetProject(ctx context.Context, id uint64) (*project.Project, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*project.Project), args.Error(1)
}
func (m *MockProjectService) GetStudentProjects(ctx context.Context, sid int64) ([]*project.Project, error) {
	args := m.Called(ctx, sid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*project.Project), args.Error(1)
}
func (m *MockProjectService) PerformAction(ctx context.Context, pid int64, action string, payload map[string]interface{}) (*project.Project, error) {
	args := m.Called(ctx, pid, action, payload)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*project.Project), args.Error(1)
}

// --- Mock gRPC Clients ---
type MockWorkflowClient struct {
	mock.Mock
	workflowv1.WorkflowServiceClient
}

func (m *MockWorkflowClient) GetActiveWorkflowByDepartment(ctx context.Context, in *workflowv1.GetActiveWorkflowByDepartmentRequest, opts ...grpc.CallOption) (*workflowv1.Workflow, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflowv1.Workflow), args.Error(1)
}
func (m *MockWorkflowClient) GetState(ctx context.Context, in *workflowv1.GetStateRequest, opts ...grpc.CallOption) (*workflowv1.State, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflowv1.State), args.Error(1)
}
func (m *MockWorkflowClient) GetNextState(ctx context.Context, in *workflowv1.GetNextStateRequest, opts ...grpc.CallOption) (*workflowv1.State, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflowv1.State), args.Error(1)
}
func (m *MockWorkflowClient) ListStateActions(ctx context.Context, in *workflowv1.ListStateActionsRequest, opts ...grpc.CallOption) (*workflowv1.ListStateActionsResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflowv1.ListStateActionsResponse), args.Error(1)
}

type MockNotificationClient struct {
	mock.Mock
	notificationv1.NotificationServiceClient
}

func (m *MockNotificationClient) SendNotification(ctx context.Context, in *notificationv1.SendNotificationRequest, opts ...grpc.CallOption) (*notificationv1.SendNotificationResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*notificationv1.SendNotificationResponse), args.Error(1)
}
