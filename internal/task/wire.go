//go:build wireinject
// +build wireinject

package task

import (
	"github.com/google/wire"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// InitializeApp инициализирует приложение task_service
func InitializeApp(cfg *Config, db *gorm.DB, logger *zap.Logger) (*Handler, func(), error) {
	wire.Build(
		NewRepository,
		NewService,
		NewHandler,
	)
	return nil, nil, nil
}
