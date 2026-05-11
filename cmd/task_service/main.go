package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lucky720s/diplomaflow/internal/task"
	"github.com/lucky720s/diplomaflow/pkg/config"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	"github.com/lucky720s/diplomaflow/pkg/metrics"
	taskv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/task/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	var cfg task.Config
	if err := config.Load("config.yaml", &cfg); err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	log := logger.New(cfg.Env)
	defer log.Sync()

	// Database connection
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database", zap.Error(err))
	}
	if err := db.AutoMigrate(&task.DeadlineNotificationRun{}); err != nil { //nolint:govet
		log.Fatal("failed to migrate task_deadline_notification_runs", zap.Error(err))
	}

	// Initialize app via Wire
	h, cleanup, err := task.InitializeApp(&cfg, db, log.Logger)
	if err != nil {
		log.Fatal("failed to initialize app", zap.Error(err))
	}
	defer cleanup()

	bgCtx, bgCancel := context.WithCancel(context.Background())
	defer bgCancel()
	h.StartBackgroundJobs(bgCtx)

	// gRPC listener
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal("failed to listen", zap.Error(err))
	}

	reg := metrics.NewRegistry("task_service")
	reg.MustRegister(metrics.GRPCCollectors()...)

	// gRPC server
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			metrics.UnaryServerInterceptor("task_service"),
			task.AuthorizationInterceptor(),
		),
	)
	taskv1.RegisterTaskServiceServer(grpcServer, h)

	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "9191"
	}
	metrics.MustServe(":"+metricsPort, reg)
	log.Info("Task metrics endpoint", zap.String("port", metricsPort))

	// Health check
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("task.v1.TaskService", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(grpcServer)

	// Start server
	go func() {
		log.Info("Task Service starting", zap.String("port", cfg.GRPCPort))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal("failed to serve", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down Task Service...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = ctx

	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	healthServer.SetServingStatus("task.v1.TaskService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	bgCancel()
	grpcServer.GracefulStop()
	log.Info("Task Service exited")
}
