//go:generate wire
//go:build wireinject
// +build wireinject

package team

import (
	"github.com/google/wire"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func InitializeApp(
	cfg *Config,
	db *gorm.DB,
	log *zap.Logger,
	authClient authv1.AuthServiceClient,
) (*EventHandler, func(), error) {
	wire.Build(
		NewRepository,
		NewService,
		NewEventHandler,
	)
	return &EventHandler{}, nil, nil
}
