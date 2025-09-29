// file: main.go
package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"

	_ "github.com/lib/pq"
	"github.com/lucky720s/diplomaflow/internal/adapters/repository/postgres"
	apphttp "github.com/lucky720s/diplomaflow/internal/delivery/http"
	"github.com/lucky720s/diplomaflow/internal/usecase"
	"github.com/lucky720s/diplomaflow/pkg/logger"
)

func main() {
	ctx := context.Background()
	log := logger.New()

	log.Info("DiplomaFlow server starting...")

	// --- Этот блок лучше вынести в пакет config в будущем ---
	// Читаем URL для подключения к БД из переменной окружения
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		postgresURL = "postgres://user:password@localhost:5432/diplomaflow?sslmode=disable"
	}
	// --------------------------------------------------------

	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Info("Successfully connected to the database")

	// --- Инициализация слоев (Dependency Injection) ---
	studentRepo := postgres.NewStudentRepo(db)
	studentUsecase := usecase.NewStudentUsecase(studentRepo, log)

	// --- Инициализация HTTP-сервера ---
	handler := apphttp.NewHandler(log, studentUsecase)
	router := handler.InitRoutes()

	serverPort := "8080"
	log.Infof("Starting HTTP server on port %s", serverPort)

	if err := http.ListenAndServe(":"+serverPort, router); err != nil {
		log.Fatalf("failed to start http server: %v", err)
	}

	log.Info("DiplomaFlow server stopped.") // Эта строка теперь не будет достигнута до остановки сервера
}
