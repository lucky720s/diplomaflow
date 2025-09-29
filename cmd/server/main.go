package main

import (
	"context"
	"database/sql"
	"github.com/lucky720s/diplomaflow/internal/adapters/repository/postgres"
	apphttp "github.com/lucky720s/diplomaflow/internal/delivery/http"
	"github.com/lucky720s/diplomaflow/internal/usecase"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	log := logger.New()
	log.Info("DiplomaFlow server starting...")

	// --- Создаем контекст, который будет отменен при получении сигнала от ОС (Ctrl+C) ---
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- Конфигурация ---
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		log.Fatal("POSTGRES_URL environment variable is not set")
	}

	// --- Подключение к БД ---
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil { // Используем PingContext с нашим корневым контекстом
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Info("Successfully connected to the database")

	// --- Инициализация слоев (Dependency Injection) ---
	studentRepo := postgres.NewStudentRepo(db)
	departmentRepo := postgres.NewDepartmentRepo(db)

	// ОБНОВЛЯЕМ ВЫЗОВ КОНСТРУКТОРА
	studentUsecase := usecase.NewStudentUsecase(studentRepo, departmentRepo, log)
	departmentUsecase := usecase.NewDepartmentUsecase(departmentRepo, log)

	handler := apphttp.NewHandler(log, studentUsecase, departmentUsecase)
	router := handler.InitRoutes()

	// --- Настройка и запуск HTTP-сервера ---
	serverPort := ":8080"
	srv := &http.Server{
		Addr:    serverPort,
		Handler: router,
	}

	// Запускаем сервер в отдельной горутине, чтобы он не блокировал основной поток
	go func() {
		log.Infof("Starting HTTP server on port %s", serverPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start http server: %v", err)
		}
	}()

	// --- Ожидание сигнала для корректной остановки ---
	<-ctx.Done() // Блокируем выполнение до тех пор, пока не будет получен сигнал (Ctrl+C)

	log.Info("Shutting down server gracefully...")

	// Создаем новый контекст с таймаутом для самой остановки
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Info("Server stopped.")
}
