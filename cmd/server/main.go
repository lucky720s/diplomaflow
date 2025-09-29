package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/lucky720s/diplomaflow/internal/adapters/repository/postgres"
	apphttp "github.com/lucky720s/diplomaflow/internal/delivery/http"
	"github.com/lucky720s/diplomaflow/internal/usecase"
	"github.com/lucky720s/diplomaflow/pkg/logger"
)

func main() {
	log := logger.New()
	log.Info("DiplomaFlow server starting...")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		log.Fatal("POSTGRES_URL environment variable is not set")
	}

	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Info("Successfully connected to the database")

	userRepo := postgres.NewUserRepo(db)
	studentRepo := postgres.NewStudentRepo(db)
	deptRepo := postgres.NewDepartmentRepo(db)

	authUsecase := usecase.NewAuthUsecase(userRepo, studentRepo, deptRepo)
	studentUsecase := usecase.NewStudentUsecase(studentRepo)
	departmentUsecase := usecase.NewDepartmentUsecase(deptRepo)

	handler := apphttp.NewHandler(log, authUsecase, studentUsecase, departmentUsecase)
	router := handler.InitRoutes()

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		log.Info("HTTP server is listening on :8080")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed to start http server: %v", err)
		}
	}()

	<-ctx.Done()

	log.Info("Shutting down server gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Info("Server stopped.")
}
