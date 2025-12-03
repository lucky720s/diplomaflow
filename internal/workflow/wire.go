//go:generate wire
//go:build wireinject
// +build wireinject

package workflow

import (
	"github.com/google/wire"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ProvideDB(cfg *Config) (*gorm.DB, func(), error) {
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}
	return db, cleanup, nil
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
