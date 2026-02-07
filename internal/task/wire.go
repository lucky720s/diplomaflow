//go:build wireinject
// +build wireinject

package task

import (
	"github.com/google/wire"
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

// InitializeApp инициализирует приложение task_service
func InitializeApp(cfg *Config, db *gorm.DB, logger *zap.Logger) (*Handler, func(), error) {
	wire.Build(
		NewRepository,
		ProvideTeamClient,
		NewService,
		NewHandler,
	)
	return nil, nil, nil
}
