//go:build wireinject
// +build wireinject

package task

import (
	"github.com/google/wire"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"
)

// ProvideTeamClient - создаёт клиент для team_service
func ProvideTeamClient(cfg *Config) (teamv1.TeamServiceClient, func(), error) {
	conn, err := grpc.NewClient(
		cfg.Services.TeamAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, err
	}
	client := teamv1.NewTeamServiceClient(conn)
	cleanup := func() { conn.Close() }
	return client, cleanup, nil
}

func ProvideNotificationClient(cfg *Config) (notificationv1.NotificationServiceClient, func(), error) {
	conn, err := grpc.NewClient(
		cfg.Services.NotificationAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, err
	}
	client := notificationv1.NewNotificationServiceClient(conn)
	return client, func() { conn.Close() }, nil
}

// InitializeApp инициализирует приложение task_service
func InitializeApp(cfg *Config, db *gorm.DB, logger *zap.Logger) (*Handler, func(), error) {
	wire.Build(
		NewRepository,             // db → Repository
		ProvideTeamClient,         // cfg → TeamServiceClient
		ProvideNotificationClient, // cfg → NotificationServiceClient
		NewService,                // Repository + TeamServiceClient + NotificationServiceClient + logger → *Service
		NewAccessChecker,          // ✅ Repository + TeamServiceClient + logger → *AccessChecker
		NewHandler,                // *Service + *AccessChecker + logger → *Handler
	)
	return nil, nil, nil
}
