//go:generate wire
//go:build wireinject
// +build wireinject

package project

import (
	"github.com/google/wire"
	"github.com/lucky720s/diplomaflow/pkg/broker"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func ProvideWorkflowClient(c workflowv1.WorkflowServiceClient) WorkflowClient {
	return c
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
