package main

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lucky720s/diplomaflow/internal/team"
	"github.com/lucky720s/diplomaflow/pkg/broker"
	"github.com/lucky720s/diplomaflow/pkg/config"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
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
	var cfg team.Config
	if err := config.Load("config.yaml", &cfg); err != nil {
		panic(err)
	}

	log := logger.New(cfg.Env)
	defer log.Sync()

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}

	// Auth client
	authConn, err := grpc.NewClient(cfg.Services.AuthAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Failed to connect to auth service", zap.Error(err))
	}
	defer authConn.Close()
	authClient := authv1.NewAuthServiceClient(authConn)

	// Workflow client
	workflowConn, err := grpc.NewClient(cfg.Services.WorkflowAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Failed to connect to workflow service", zap.Error(err))
	}
	defer workflowConn.Close()
	workflowClient := workflowv1.NewWorkflowServiceClient(workflowConn)

	// Notification client
	notificationConn, err := grpc.NewClient(cfg.Services.NotificationAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Failed to connect to notification service", zap.Error(err))
	}
	defer notificationConn.Close()
	notificationClient := notificationv1.NewNotificationServiceClient(notificationConn)

	// Initialize app (теперь с 6 аргументами)
	app, cleanup, err := team.InitializeApp(
		&cfg,
		db,
		log.Logger,
		authClient,
		workflowClient,
		notificationClient,
	)
	if err != nil {
		log.Fatal("failed to initialize app", zap.Error(err))
	}
	defer cleanup()

	h := app.Handler
	eventHandler := app.EventHandler

	ctx, cancel := context.WithCancel(context.Background())

	if cfg.Kafka.Enabled {
		log.Info("Kafka ENABLED - starting consumer...")

		brokers := strings.Split(cfg.Kafka.Brokers, ",")
		groupID := cfg.Kafka.GroupID
		if groupID == "" {
			groupID = "team-service-group"
		}

		kafkaConsumer, consumerErr := broker.NewConsumerWithRetry(brokers, groupID, log.Logger, broker.DefaultRetryConfig())
		if consumerErr != nil {
			log.Fatal("Failed to create kafka consumer", zap.Error(consumerErr))
		}
		defer kafkaConsumer.Close()

		handlerFn := func(ctx context.Context, event broker.Event) error {
			var payload team.ProjectCreatedEvent
			if unmarshalErr := json.Unmarshal(event.Payload, &payload); unmarshalErr != nil {
				return broker.Permanent(unmarshalErr)
			}
			return eventHandler.HandleProjectCreated(ctx, payload)
		}

		go func() {
			log.Info("Starting Kafka consumer",
				zap.Strings("topics", []string{"project-events"}),
				zap.String("group_id", groupID))
			kafkaConsumer.Start(ctx, []string{"project-events"}, handlerFn)
		}()
	} else {
		log.Warn("Kafka DISABLED - team_service will NOT receive project events automatically")
		log.Warn("Projects must be assigned to teams manually via AssignProject RPC")
	}

	// gRPC server
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal("failed to listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	teamv1.RegisterTeamServiceServer(grpcServer, h)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("team.v1.TeamService", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(grpcServer)

	go func() {
		log.Info("Team Service starting",
			zap.String("port", cfg.GRPCPort),
			zap.Bool("kafka_enabled", cfg.Kafka.Enabled))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal("failed to serve", zap.Error(err))
		}
	}()

	log.Info("Team Service started")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down...")
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	healthServer.SetServingStatus("team.v1.TeamService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	cancel()
	grpcServer.GracefulStop()
	time.Sleep(300 * time.Millisecond)

	log.Info("Team Service exited")
}
