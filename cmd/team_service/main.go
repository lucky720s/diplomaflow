package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lucky720s/diplomaflow/internal/team"
	"github.com/lucky720s/diplomaflow/pkg/config"
	grpcpkg "github.com/lucky720s/diplomaflow/pkg/grpc"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
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

func dial(addr string) (*grpc.ClientConn, error) {
	return grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(grpcpkg.DefaultRetryServiceConfig()),
		grpc.WithUnaryInterceptor(grpcpkg.TimeoutInterceptor(10*time.Second)),
	)
}
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
	authConn, err := dial(cfg.Services.AuthAddr)
	if err != nil {
		log.Fatal("Failed to connect to auth service", zap.Error(err))
	}
	defer authConn.Close()
	authClient := authv1.NewAuthServiceClient(authConn)

	// Workflow client
	workflowConn, err := dial(cfg.Services.WorkflowAddr)
	if err != nil {
		log.Fatal("Failed to connect to workflow service", zap.Error(err))
	}
	defer workflowConn.Close()
	workflowClient := workflowv1.NewWorkflowServiceClient(workflowConn)

	// Notification client
	notificationConn, err := dial(cfg.Services.NotificationAddr)
	if err != nil {
		log.Fatal("Failed to connect to notification service", zap.Error(err))
	}
	defer notificationConn.Close()
	notificationClient := notificationv1.NewNotificationServiceClient(notificationConn)

	// Admin client
	adminConn, err := dial(cfg.Services.AdminAddr)
	if err != nil {
		log.Fatal("Failed to connect to admin service", zap.Error(err))
	}
	defer adminConn.Close()
	adminClient := adminv1.NewAdminServiceClient(adminConn)

	// Initialize app (теперь с 6 аргументами)
	app, cleanup, err := team.InitializeApp(
		&cfg,
		db,
		log.Logger,
		authClient,
		workflowClient,
		notificationClient,
		adminClient,
	)
	if err != nil {
		log.Fatal("failed to initialize app", zap.Error(err))
	}
	defer cleanup()

	h := app.Handler

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
			zap.String("port", cfg.GRPCPort))
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

	grpcServer.GracefulStop()
	time.Sleep(300 * time.Millisecond)

	log.Info("Team Service exited")
}
