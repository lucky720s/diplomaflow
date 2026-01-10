//go:generate wire
//go:build wireinject
// +build wireinject

package workflow

import (
	"github.com/google/wire"
	"github.com/lucky720s/diplomaflow/pkg/database"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func ProvideDB(cfg *Config) (*gorm.DB, func(), error) {
	return database.NewConnection(cfg.Database.DSN)
}

// InitializeApp builds base workflow CRUD handler (no runtime deps here).
func InitializeApp(cfg *Config, log *zap.Logger) (*Handler, func(), error) {
	wire.Build(
		ProvideDB,
		NewRepository,
		NewService,
		NewHandler,
	)
	return &Handler{}, nil, nil
}
