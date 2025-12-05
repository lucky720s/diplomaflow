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

func ProvideProcessorRegistry() *ProcessorRegistry {
	registry := NewProcessorRegistry()
	registry.Register("SELECT_SUPERVISOR", &SupervisorSelectionHandler{})
	registry.Register("UPLOAD_FILE", &DocumentUploadHandler{})
	registry.Register("APPROVE", &ApprovalHandler{})
	registry.Register("REJECT", &ApprovalHandler{})
	return registry
}

func InitializeApp(
	cfg *Config,
	db *gorm.DB,
	log *zap.Logger,
	kafkaProducer *broker.Producer,
	wfClient workflowv1.WorkflowServiceClient,
) (*Handler, func(), error) {
	wire.Build(
		NewRepository,
		ProvideProcessorRegistry,
		NewService,
		NewHandler,
	)
	return &Handler{}, nil, nil
}
