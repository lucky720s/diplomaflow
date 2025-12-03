package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lucky720s/diplomaflow/internal/team"
	"github.com/lucky720s/diplomaflow/pkg/broker"
	"github.com/lucky720s/diplomaflow/pkg/config"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	var cfg team.Config
	if err := config.Load("config.yaml", &cfg); err != nil {
		panic(err)
	}

	log := logger.New(cfg.Env)
	defer log.Sync()

	app, cleanup, err := team.InitializeApp(&cfg, log)
	if err != nil {
		log.Fatal("failed to initialize app", zap.Error(err))
	}
	defer cleanup()

	ctx, cancelCtx := context.WithCancel(context.Background())
	app.Consumer.Start(ctx, []string{"project-events"}, func(ctx context.Context, event broker.Event) error {
		if event.Type == "ProjectCreated" {
			return app.EventHandler.HandleProjectCreated(event)
		}
		return nil
	})

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal("listen error", zap.Error(err))
	}
	grpcServer := grpc.NewServer()
	teamv1.RegisterTeamServiceServer(grpcServer, app.Handler)

	go func() {
		log.Info("Team Service starting", zap.String("port", cfg.GRPCPort))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal("serve error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down Team Service...")

	cancelCtx()

	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	grpcServer.GracefulStop()

	log.Info("Team Service exited")
}
