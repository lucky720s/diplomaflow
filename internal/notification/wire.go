//go:generate wire
//go:build wireinject
// +build wireinject

package notification

import (
	"github.com/google/wire"
	"github.com/lucky720s/diplomaflow/pkg/database"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	"gorm.io/gorm"
)

func ProvideDB(cfg *Config) (*gorm.DB, func(), error) {
	return database.NewConnection(cfg.Database.DSN)
}

func InitializeApp(cfg *Config, log *logger.Logger) (*Handler, func(), error) {
	wire.Build(
		ProvideDB,
		NewRepository,
		NewService,
		NewHandler,
	)
	return &Handler{}, nil, nil
}
