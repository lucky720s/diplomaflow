package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lucky720s/diplomaflow/internal/project"
	"github.com/lucky720s/diplomaflow/pkg/config"
	grpcpkg "github.com/lucky720s/diplomaflow/pkg/grpc"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	"github.com/lucky720s/diplomaflow/pkg/metrics"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func dial(addr string) (*grpc.ClientConn, error) {
	return grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(grpcpkg.DefaultRetryServiceConfig()),
		grpc.WithUnaryInterceptor(grpcpkg.TimeoutInterceptor(10*time.Second)),
	)
}
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

	// Workflow client
	wfConn, err := dial(cfg.Services.WorkflowAddr)
	if err != nil {
		log.Fatal("Failed to connect to workflow service", zap.Error(err))
	}
	defer wfConn.Close()
	wfClient := workflowv1.NewWorkflowServiceClient(wfConn)

	app, cleanup, err := project.InitializeApp(&cfg, db, log.Logger, wfClient)
	if err != nil {
		log.Fatal("failed to initialize app", zap.Error(err))
	}
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	go app.DeadlineScheduler.Start(ctx)

	// gRPC server
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal("listen error", zap.Error(err))
	}

	reg := metrics.NewRegistry("project_service")
	reg.MustRegister(metrics.GRPCCollectors()...)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(metrics.UnaryServerInterceptor("project_service")),
	)
	projectv1.RegisterProjectServiceServer(grpcServer, app.Handler)

	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "9083"
	}
	metrics.MustServe(":"+metricsPort, reg)
	log.Info("Project metrics endpoint", zap.String("port", metricsPort))

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("project.v1.ProjectService", grpc_health_v1.HealthCheckResponse_SERVING)

	go func() {
		log.Info("Project Service starting",
			zap.String("port", cfg.GRPCPort))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal("serve error", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down...")
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	healthServer.SetServingStatus("project.v1.ProjectService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	cancel()

	app.DeadlineScheduler.Stop()
	time.Sleep(500 * time.Millisecond)
	grpcServer.GracefulStop()

	log.Info("Project Service exited")
}
