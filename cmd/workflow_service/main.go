package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lucky720s/diplomaflow/internal/workflow"
	"github.com/lucky720s/diplomaflow/internal/workflow/engine"
	"github.com/lucky720s/diplomaflow/internal/workflow/outboxpoller"
	"github.com/lucky720s/diplomaflow/internal/workflow/plugins/builtin"
	"github.com/lucky720s/diplomaflow/internal/workflow/postcommit"
	"github.com/lucky720s/diplomaflow/internal/workflow/runtimegrpc"
	"github.com/lucky720s/diplomaflow/pkg/config"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
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

	teamConn, err := grpc.NewClient(cfg.Services.TeamAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("failed to connect team service", zap.Error(err))
	}
	defer teamConn.Close()
	teamClient := teamv1.NewTeamServiceClient(teamConn)

	builtin.RegisterAll(notifClient, teamClient)
	log.Info("Builtin plugins registered", zap.Strings("plugins", builtin.RegisteredPlugins()))

	repo := workflow.NewRepository(db)
	svc := workflow.NewService(repo, log.Logger)
	base := workflow.NewHandler(svc, log.Logger)

	// NEW: engine gets team client (TEAM_FORMED validation)
	eng := engine.NewWorkflowEngine(repo, teamClient, log.Logger)

	h := runtimegrpc.New(base, eng, projectClient, log.Logger)

	pcWorker := postcommit.NewWorker(db, projectClient, log.Logger)
	pcGRPC := postcommit.NewGRPCServer(pcWorker, log.Logger)

	poller, err := outboxpoller.New(db, pcWorker, log.Logger, outboxpoller.Config{
		Enabled:           cfg.OutboxPoller.Enabled,
		Topic:             cfg.OutboxPoller.Topic,
		IntervalSeconds:   cfg.OutboxPoller.IntervalSeconds,
		BatchSize:         cfg.OutboxPoller.BatchSize,
		Table:             cfg.OutboxPoller.Table,
		IDColumn:          cfg.OutboxPoller.IDColumn,
		TopicColumn:       cfg.OutboxPoller.TopicColumn,
		StatusColumn:      cfg.OutboxPoller.StatusColumn,
		EventTypeColumn:   cfg.OutboxPoller.EventTypeColumn,
		PayloadColumn:     cfg.OutboxPoller.PayloadColumn,
		ProcessedAtColumn: cfg.OutboxPoller.ProcessedAtColumn,
		PendingStatus:     cfg.OutboxPoller.PendingStatus,
		ProcessedStatus:   cfg.OutboxPoller.ProcessedStatus,
	})
	if err != nil {
		log.Fatal("failed to init outbox poller", zap.Error(err))
	}

	pollerCtx, pollerCancel := context.WithCancel(context.Background())
	go poller.Start(pollerCtx)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal("failed to listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	workflowv1.RegisterWorkflowServiceServer(grpcServer, h)
	workflowv1.RegisterWorkflowActionsServiceServer(grpcServer, pcGRPC)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("workflow.v1.WorkflowService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("workflow.v1.WorkflowActionsService", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(grpcServer)

	go func() {
		log.Info("Workflow Service starting", zap.String("port", cfg.GRPCPort))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal("failed to serve", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Info("Shutdown signal received")
	pollerCancel()

	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	healthServer.SetServingStatus("workflow.v1.WorkflowService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	healthServer.SetServingStatus("workflow.v1.WorkflowActionsService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	time.Sleep(300 * time.Millisecond)
	grpcServer.GracefulStop()

	log.Info("Workflow Service stopped")
}
