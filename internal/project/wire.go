//go:generate wire
//go:build wireinject
// +build wireinject

package project

import (
	"github.com/google/wire"
	"github.com/lucky720s/diplomaflow/pkg/broker"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"
)

type App struct {
	Handler           *Handler
	ActionExecutor    *StateActionExecutor
	DeadlineScheduler *DeadlineScheduler
}

func NewApp(h *Handler, executor *StateActionExecutor, repo Repository, logger *zap.Logger) *App {
	scheduler := NewDeadlineScheduler(repo, executor, logger)
	return &App{
		Handler:           h,
		ActionExecutor:    executor,
		DeadlineScheduler: scheduler,
	}
}
func ProvideProcessorRegistry() *ProcessorRegistry {
	registry := NewProcessorRegistry()
	registry.Register("TEAM_FORMED", &TeamFormedHandler{})
	registry.Register("SELECT_SUPERVISOR", &SelectSupervisorHandler{})
	registry.Register("TOPIC_APPROVED", &TopicApprovedHandler{})
	registry.Register("UPLOAD_TASK", &UploadTaskHandler{})
	registry.Register("TASK_UPLOADED", &TaskUploadedHandler{})
	registry.Register("APPROVE", &ApproveHandler{})
	registry.Register("REJECT", &RejectHandler{})
	return registry
}

func ProvideNotificationClient(cfg *Config) (notificationv1.NotificationServiceClient, func(), error) {
	conn, err := grpc.NewClient(cfg.Services.NotificationAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { conn.Close() }
	return notificationv1.NewNotificationServiceClient(conn), cleanup, nil
}

func ProvideStateActionExecutor(
	wfClient workflowv1.WorkflowServiceClient,
	notifClient notificationv1.NotificationServiceClient,
	logger *zap.Logger,
) *StateActionExecutor {
	return NewStateActionExecutor(wfClient, notifClient, logger)
}

func InitializeApp(
	cfg *Config,
	db *gorm.DB,
	log *zap.Logger,
	kafkaProducer *broker.Producer,
	wfClient workflowv1.WorkflowServiceClient,
) (*App, func(), error) {
	wire.Build(
		NewRepository,
		ProvideProcessorRegistry,
		ProvideNotificationClient,
		ProvideStateActionExecutor,
		NewService,
		wire.Bind(new(ProjectUseCase), new(*Service)),
		NewHandler,
		NewApp,
	)
	return &App{}, nil, nil
}
