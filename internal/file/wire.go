//go:generate wire
//go:build wireinject
// +build wireinject

package file

import (
	"github.com/google/wire"
	"github.com/lucky720s/diplomaflow/pkg/logger"
)

func InitializeApp(cfg *Config, log *logger.Logger) (*Handler, func(), error) {
	wire.Build(
		NewService,
		NewHandler,
	)
	return &Handler{}, nil, nil
}
