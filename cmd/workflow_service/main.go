package main

import (
	"context"
	"log"

	"github.com/lucky720s/diplomaflow/internal/workflow"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	rkboot "github.com/rookie-ninja/rk-boot/v2"
	rkpostgres "github.com/rookie-ninja/rk-db/postgres"
	rkgrpc "github.com/rookie-ninja/rk-grpc/v2/boot"
	"google.golang.org/grpc"
)

func main() {
	boot := rkboot.NewBoot(rkboot.WithBootConfigPath("boot.yaml", nil))

	postgresEntry := rkpostgres.GetPostgresEntry("workflow-conn")
	postgresEntry.Bootstrap(context.Background())
	if postgresEntry == nil {
		log.Fatal("Missing 'workflow-conn' in boot.yaml")
	}
	db := postgresEntry.GetDB("diplomaflow")
	if db == nil {
		log.Fatal("Database 'diplomaflow' not found")
	}

	repo := workflow.NewRepository(db)
	svc := workflow.NewService(repo)
	handler := workflow.NewHandler(svc)

	grpcEntry := rkgrpc.GetGrpcEntry("workflow-service")
	grpcEntry.AddRegFuncGrpc(func(s *grpc.Server) {
		workflowv1.RegisterWorkflowServiceServer(s, handler)
	})

	boot.Bootstrap(context.Background())
	boot.WaitForShutdownSig(context.Background())
}
