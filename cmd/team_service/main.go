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
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
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

	authConn, err := grpc.NewClient(cfg.Services.AuthAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Failed to connect to auth service", zap.Error(err))
	}
	defer authConn.Close()
	authClient := authv1.NewAuthServiceClient(authConn)

	app, cleanup, err := team.InitializeApp(&cfg, db, log.Logger, authClient)
	if err != nil {
		log.Fatal("failed to initialize app", zap.Error(err))
	}
	defer cleanup()

	h := app.Handler
	eventHandler := app.EventHandler

	brokers := strings.Split(cfg.Kafka.Brokers, ",")
	groupID := cfg.Kafka.GroupID
	if groupID == "" {
		groupID = "team-service-group"
	}

	kafkaConsumer, err := broker.NewConsumerWithRetry(brokers, groupID, log.Logger, broker.DefaultRetryConfig())
	if err != nil {
		log.Fatal("Failed to create kafka consumer", zap.Error(err))
	}
	defer kafkaConsumer.Close()

	ctx, cancel := context.WithCancel(context.Background())

	handlerFn := func(ctx context.Context, event broker.Event) error {
		var payload team.ProjectCreatedEvent
		if unmarshalErr := json.Unmarshal(event.Payload, &payload); unmarshalErr != nil {
			return broker.Permanent(unmarshalErr)
		}
		return eventHandler.HandleProjectCreated(ctx, payload)
	}

	go func() {
		log.Info("Starting Kafka consumer", zap.Strings("topics", []string{"project-events"}), zap.String("group_id", groupID))
		kafkaConsumer.Start(ctx, []string{"project-events"}, handlerFn)
	}()

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal("failed to listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	teamv1.RegisterTeamServiceServer(grpcServer, h)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("team.TeamService", grpc_health_v1.HealthCheckResponse_SERVING)
	reflection.Register(grpcServer)

	go func() {
		log.Info("Team Service starting", zap.String("port", cfg.GRPCPort))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal("failed to serve", zap.Error(err))
		}
	}()

	log.Info("Team Service started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down...")
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	healthServer.SetServingStatus("team.TeamService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	cancel()
	grpcServer.GracefulStop()
	time.Sleep(300 * time.Millisecond)
	log.Info("Team Service exited")
}
