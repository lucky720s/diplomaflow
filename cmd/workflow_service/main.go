package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lucky720s/diplomaflow/internal/workflow"
	"github.com/lucky720s/diplomaflow/internal/workflow/engine"
	"github.com/lucky720s/diplomaflow/internal/workflow/plugins/builtin"
	"github.com/lucky720s/diplomaflow/internal/workflow/runtimegrpc"
	"github.com/lucky720s/diplomaflow/pkg/config"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	var cfg workflow.Config
	if err := config.Load("config.yaml", &cfg); err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	log := logger.New(cfg.Env)
	defer log.Sync()

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}

	// gRPC clients
	notifConn, err := grpc.NewClient(cfg.Services.NotificationAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("failed to connect to notification service", zap.Error(err))
	}
	defer notifConn.Close()
	notifClient := notificationv1.NewNotificationServiceClient(notifConn)

	projectConn, err := grpc.NewClient(cfg.Services.ProjectAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("failed to connect to project service", zap.Error(err))
	}
	defer projectConn.Close()
	projectClient := projectv1.NewProjectServiceClient(projectConn)

	// Core workflow components
	repo := workflow.NewRepository(db)
	svc := workflow.NewService(repo, log.Logger)
	eng := engine.NewWorkflowEngine(repo, log.Logger)

	// Plugins
	builtin.RegisterAll(notifClient)
	log.Info("Builtin plugins registered", zap.Strings("plugins", builtin.RegisteredPlugins()))

	// Runtime gRPC handler (no import cycle)
	h := runtimegrpc.New(svc, eng, projectClient, log.Logger)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal("failed to listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	workflowv1.RegisterWorkflowServiceServer(grpcServer, h)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("workflow.v1.WorkflowService", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(grpcServer)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = strings.Split(cfg.Kafka.Brokers, ",")
	_ = ctx

	go func() {
		log.Info("Workflow Service starting", zap.String("port", cfg.GRPCPort))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal("failed to serve", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down Workflow Service...")
	healthServer.SetServingStatus("workflow.v1.WorkflowService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	cancel()
	time.Sleep(150 * time.Millisecond)
	grpcServer.GracefulStop()
	log.Info("Workflow Service exited")
}
