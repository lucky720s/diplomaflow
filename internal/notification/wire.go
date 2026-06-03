//go:generate wire
//go:build wireinject
// +build wireinject

package notification

import (
	"github.com/google/wire"
	"github.com/lucky720s/diplomaflow/pkg/database"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	"github.com/lucky720s/diplomaflow/pkg/realtime"
	"gorm.io/gorm"
)

func ProvideDB(cfg *Config) (*gorm.DB, func(), error) {
	return database.NewConnection(cfg.Database.DSN)
}

func ProvideRealtimePublisher(cfg *Config) (realtime.Publisher, func(), error) {
	return realtime.NewPublisher(cfg.RedisAddr)
}

func InitializeApp(cfg *Config, log *logger.Logger) (*Handler, func(), error) {
	wire.Build(
		ProvideDB,
		NewRepository,
		NewPusher,
		ProvideRealtimePublisher,
		NewService,
		wire.Bind(new(NotificationUseCase), new(*Service)),
		NewHandler,
	)
	return &Handler{}, nil, nil
}
