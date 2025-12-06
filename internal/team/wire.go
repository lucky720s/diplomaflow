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

func InitializeApp(
	cfg *Config,
	db *gorm.DB,
	log *zap.Logger,
	authClient authv1.AuthServiceClient,
) *App {
	wire.Build(
		NewRepository,
		NewService,
		NewEventHandler,
		NewHandler,
		NewApp,
	)
	return &App{}
}
