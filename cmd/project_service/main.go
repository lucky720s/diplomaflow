package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lucky720s/diplomaflow/internal/project"
	"github.com/lucky720s/diplomaflow/pkg/broker"
	"github.com/lucky720s/diplomaflow/pkg/config"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	var cfg project.Config
	if err := config.Load("config.yaml", &cfg); err != nil {
		panic(err)
	}

	log := logger.New(cfg.Env)
	defer log.Sync()

	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}

	brokers := strings.Split(cfg.Kafka.Brokers, ",")
	kafkaProducer, err := broker.NewProducer(brokers)
	if err != nil {
		log.Fatal("Failed to create kafka producer", zap.Error(err))
	}
	defer kafkaProducer.Close()

	wfConn, err := grpc.NewClient(cfg.Services.WorkflowAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Failed to connect to workflow service", zap.Error(err))
	}
	defer wfConn.Close()
	wfClient := workflowv1.NewWorkflowServiceClient(wfConn)

	app, cleanup, err := project.InitializeApp(&cfg, db, log.Logger, kafkaProducer, wfClient)
	if err != nil {
		log.Fatal("failed to initialize app", zap.Error(err))
	}
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	go app.OutboxProcessor.Start(ctx)
	go app.DeadlineScheduler.Start(ctx)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal("listen error", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	projectv1.RegisterProjectServiceServer(grpcServer, app.Handler)

	go func() {
		log.Info("Project Service starting", zap.String("port", cfg.GRPCPort))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal("serve error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down...")
	cancel()
	app.OutboxProcessor.Stop()
	app.DeadlineScheduler.Stop()
	time.Sleep(500 * time.Millisecond)
	grpcServer.GracefulStop()
	log.Info("Project Service exited")
}
