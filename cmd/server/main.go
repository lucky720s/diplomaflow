package main

import (
	"context"
	"github.com/lucky720s/diplomaflow/pkg/logger"
)

func main() {
	// Создаём root context (будем передавать вниз по слоям)
	ctx := context.Background()

	// Инициализируем логгер
	log := logger.New()

	log.Info("DiplomaFlow server starting...")

	// TODO: инициализация конфигов, базы, API
	// Для начала просто завершим
	log.Info("DiplomaFlow server stopped.")
}
