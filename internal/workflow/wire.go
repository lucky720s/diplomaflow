//go:generate wire
//go:build wireinject
// +build wireinject

package workflow

import (
	"github.com/google/wire"
	"github.com/lucky720s/diplomaflow/pkg/database"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func ProvideDB(cfg *Config) (*gorm.DB, func(), error) {
	return database.NewConnection(cfg.Database.DSN)
}

func ProvideLogger(log *logger.Logger) *zap.Logger {
	return log.Logger
}

func ProvideRepository(db *gorm.DB) Repository {
	return NewRepository(db)
}

func ProvideService(repo Repository, logger *zap.Logger) *Service {
	return NewService(repo, logger)
}

func ProvideHandler(service *Service, logger *zap.Logger) *Handler {
	return NewHandler(service, logger)
}

func InitializeApp(cfg *Config, log *logger.Logger) (*Handler, func(), error) {
	wire.Build(
		ProvideDB,
		ProvideLogger,
		ProvideRepository,
		ProvideService,
		ProvideHandler,
	)
	return &Handler{}, nil, nil
}
