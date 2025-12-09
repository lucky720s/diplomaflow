package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lucky720s/diplomaflow/internal/form"
	"github.com/lucky720s/diplomaflow/pkg/config"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	formv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/form/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	var cfg form.Config
	if err := config.Load("config.yaml", &cfg); err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	log := logger.New(cfg.Env)
	defer log.Sync()
	h, cleanup, err := form.InitializeApp(&cfg, log)
	if err != nil {
		log.Fatal("failed to initialize app", zap.Error(err))
	}
	defer cleanup()

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal("failed to listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	formv1.RegisterFormServiceServer(grpcServer, h)
	reflection.Register(grpcServer)

	go func() {
		log.Info("Form Service starting", zap.String("port", cfg.GRPCPort))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal("failed to serve", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down Form Service...")
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	grpcServer.GracefulStop()
	log.Info("Form Service exited")
}
