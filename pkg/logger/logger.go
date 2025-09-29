package logger

import (
	"go.uber.org/zap"
)

type Logger struct {
	*zap.SugaredLogger
}

func New() *Logger {
	logger, _ := zap.NewProduction() // можно заменить на zap.NewDevelopment()
	defer logger.Sync()

	return &Logger{logger.Sugar()}
}
