//go:generate wire
//go:build wireinject
// +build wireinject

package project

import (
	"context"

	"github.com/google/wire"
	"github.com/lucky720s/diplomaflow/pkg/broker"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func ProvideWorkflowClient(c workflowv1.WorkflowServiceClient) WorkflowClient {
	// workflowv1 client implements the methods we need; we call without opts.
	return &workflowClientWrapper{c: c}
}

// wrapper converts ...grpc.CallOption to ...interface{} usage (we ignore opts)
type workflowClientWrapper struct {
	c workflowv1.WorkflowServiceClient
}

func (w *workflowClientWrapper) GetActiveWorkflowByDepartment(ctx context.Context, in *workflowv1.GetActiveWorkflowByDepartmentRequest, _ ...interface{}) (*workflowv1.Workflow, error) {
	return w.c.GetActiveWorkflowByDepartment(ctx, in)
}
func (w *workflowClientWrapper) GetAvailableTransitions(ctx context.Context, in *workflowv1.GetAvailableTransitionsRequest, _ ...interface{}) (*workflowv1.GetAvailableTransitionsResponse, error) {
	return w.c.GetAvailableTransitions(ctx, in)
}
func (w *workflowClientWrapper) ExecuteTransition(ctx context.Context, in *workflowv1.ExecuteTransitionRequest, _ ...interface{}) (*workflowv1.ExecuteTransitionResponse, error) {
	return w.c.ExecuteTransition(ctx, in)
}

func InitializeApp(
	cfg *Config,
	db *gorm.DB,
	log *zap.Logger,
	kafkaProducer *broker.Producer,
	wfClient workflowv1.WorkflowServiceClient,
) (*App, func(), error) {
	wire.Build(
		ProvideWorkflowClient,

		NewRepository,
		NewService,
		wire.Bind(new(ProjectUseCase), new(*Service)),

		NewHandler,

		NewDeadlineScheduler,
		NewOutboxProcessor,

		NewApp,
	)
	return &App{}, nil, nil
}
