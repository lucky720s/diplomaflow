//go:build wireinject
// +build wireinject

package team

import (
	"github.com/google/wire"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type App struct {
	Handler *Handler
}

func InitializeApp(
	cfg *Config,
	db *gorm.DB,
	logger *zap.Logger,
	authClient authv1.AuthServiceClient,
	workflowClient workflowv1.WorkflowServiceClient, // <-- ДОБАВЛЕНО
	notificationClient notificationv1.NotificationServiceClient, // <-- ДОБАВЛЕНО
) (*App, func(), error) {
	wire.Build(
		NewRepository,
		NewService,
		NewHandler,
		wire.Struct(new(App), "*"),
	)
	return nil, nil, nil
}
