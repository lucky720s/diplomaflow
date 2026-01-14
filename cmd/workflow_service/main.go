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
	"github.com/lucky720s/diplomaflow/internal/workflow/postcommit"
	"github.com/lucky720s/diplomaflow/internal/workflow/runtimegrpc"
	"github.com/lucky720s/diplomaflow/pkg/broker"
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
	defer func() { _ = log.Sync() }()

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect db", zap.Error(err))
	}

	// gRPC clients
	notifConn, err := grpc.NewClient(cfg.Services.NotificationAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("failed to connect notification service", zap.Error(err))
	}
	defer notifConn.Close()
	notifClient := notificationv1.NewNotificationServiceClient(notifConn)

	projectConn, err := grpc.NewClient(cfg.Services.ProjectAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("failed to connect project service", zap.Error(err))
	}
	defer projectConn.Close()
	projectClient := projectv1.NewProjectServiceClient(projectConn)

	// plugins
	builtin.RegisterAll(notifClient)
	log.Info("Builtin plugins registered", zap.Strings("plugins", builtin.RegisteredPlugins()))

	// workflow core
	repo := workflow.NewRepository(db)
	svc := workflow.NewService(repo, log.Logger)
	base := workflow.NewHandler(svc, log.Logger)
	eng := engine.NewWorkflowEngine(repo, log.Logger)
	h := runtimegrpc.New(base, eng, projectClient, log.Logger)

	// Kafka consumer: workflow-actions (POST actions + deadlines)
	brokers := strings.Split(cfg.Kafka.Brokers, ",")
	groupID := cfg.Kafka.GroupID
	if groupID == "" {
		groupID = "workflow-service-group"
	}

	consumer, err := broker.NewConsumer(brokers, groupID, log.Logger)
	if err != nil {
		log.Fatal("failed to create kafka consumer", zap.Error(err))
	}
	defer consumer.Close()

	worker := postcommit.NewWorker(db, projectClient, log.Logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		consumer.Start(ctx, []string{"workflow-actions"}, worker.HandleEvent)
	}()

	// gRPC server
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal("failed to listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	workflowv1.RegisterWorkflowServiceServer(grpcServer, h)

	// gRPC health
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	// ВАЖНО: выставляем и общий статус "", и статус конкретного сервиса
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("workflow.v1.WorkflowService", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(grpcServer)

	// Start serving
	go func() {
		log.Info("Workflow Service starting", zap.String("port", cfg.GRPCPort))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal("failed to serve", zap.Error(err))
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Info("Shutdown signal received")

	// Mark NOT_SERVING before stopping (so orchestrator stops routing)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	healthServer.SetServingStatus("workflow.v1.WorkflowService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// Stop background workers
	cancel()

	// Give some time for background goroutines to finish
	time.Sleep(300 * time.Millisecond)

	grpcServer.GracefulStop()
	log.Info("Workflow Service stopped")
}
