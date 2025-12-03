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
	"github.com/lucky720s/diplomaflow/pkg/logger"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	var cfg project.Config
	if err := config.Load("config.yaml", &cfg); err != nil {
		panic(err)
	}

	log := logger.New(cfg.Env)
	defer log.Sync()

	h, cleanup, err := project.InitializeApp(&cfg, log)
	if err != nil {
		log.Fatal("failed to initialize app", zap.Error(err))
	}
	defer cleanup()

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal("listen error", zap.Error(err))
	}
	grpcServer := grpc.NewServer()
	projectv1.RegisterProjectServiceServer(grpcServer, h)

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
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcServer.GracefulStop()
	log.Info("Project Service exited")
}
