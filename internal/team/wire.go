//go:generate wire
//go:build wireinject
// +build wireinject

package team

import (
	"github.com/google/wire"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type App struct {
	Handler      *Handler
	EventHandler *EventHandler
}

func NewApp(h *Handler, eh *EventHandler) *App {
	return &App{
		Handler:      h,
		EventHandler: eh,
	}
}
func ProvideNotificationClient(cfg *Config) (notificationv1.NotificationServiceClient, func(), error) {
	conn, err := grpc.NewClient(cfg.Services.NotificationAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { conn.Close() }
	return notificationv1.NewNotificationServiceClient(conn), cleanup, nil
}

func InitializeApp(
	cfg *Config,
	db *gorm.DB,
	log *zap.Logger,
	authClient authv1.AuthServiceClient,
) (*App, func(), error) {
	wire.Build(
		NewRepository,
		NewService,
		NewEventHandler,
		NewHandler,
		NewApp,
		ProvideNotificationClient,
	)
	return &App{}, nil, nil
}
