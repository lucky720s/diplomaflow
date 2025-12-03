package main

import (
	"context"
	"log"

	"github.com/lucky720s/diplomaflow/internal/university"
	universityv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/university/v1"
	rkboot "github.com/rookie-ninja/rk-boot/v2"
	rkpostgres "github.com/rookie-ninja/rk-db/postgres"
	rkgrpc "github.com/rookie-ninja/rk-grpc/v2/boot"
	"google.golang.org/grpc"
)

func main() {
	boot := rkboot.NewBoot(rkboot.WithBootConfigPath("boot.yaml", nil))

	postgresEntry := rkpostgres.GetPostgresEntry("university-conn")
	postgresEntry.Bootstrap(context.Background())
	if postgresEntry == nil {
		log.Fatal("Missing 'university-conn' in boot.yaml")
	}
	db := postgresEntry.GetDB("diplomaflow")
	if db == nil {
		log.Fatal("Database 'diplomaflow' not found")
	}

	repo := university.NewRepository(db)
	svc := university.NewService(repo)
	handler := university.NewHandler(svc)

	grpcEntry := rkgrpc.GetGrpcEntry("university-service")
	grpcEntry.AddRegFuncGrpc(func(s *grpc.Server) {
		universityv1.RegisterUniversityServiceServer(s, handler)
	})

	boot.Bootstrap(context.Background())
	boot.WaitForShutdownSig(context.Background())
}
