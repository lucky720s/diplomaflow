package main

import (
	"context"
	"log"

	"github.com/lucky720s/diplomaflow/internal/role"
	rolev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/role/v1"
	rkboot "github.com/rookie-ninja/rk-boot/v2"
	rkpostgres "github.com/rookie-ninja/rk-db/postgres"
	rkgrpc "github.com/rookie-ninja/rk-grpc/v2/boot"
	"google.golang.org/grpc"
)

func main() {
	boot := rkboot.NewBoot(rkboot.WithBootConfigPath("boot.yaml", nil))

	postgresEntry := rkpostgres.GetPostgresEntry("role-conn")
	postgresEntry.Bootstrap(context.Background())
	if postgresEntry == nil {
		log.Fatal("Missing 'role-conn' in boot.yaml")
	}
	db := postgresEntry.GetDB("diplomaflow")
	if db == nil {
		log.Fatal("Database 'diplomaflow' not found")
	}

	repo := role.NewRepository(db)
	svc := role.NewService(repo)
	handler := role.NewHandler(svc)

	grpcEntry := rkgrpc.GetGrpcEntry("role-service")
	grpcEntry.AddRegFuncGrpc(func(s *grpc.Server) {
		rolev1.RegisterRoleServiceServer(s, handler)
	})

	boot.Bootstrap(context.Background())
	boot.WaitForShutdownSig(context.Background())
}
