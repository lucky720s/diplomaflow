package main

import (
	"context"
	"log"
	"os"

	"github.com/lucky720s/diplomaflow/internal/project"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	rkboot "github.com/rookie-ninja/rk-boot/v2"
	rkpostgres "github.com/rookie-ninja/rk-db/postgres"
	rkgrpc "github.com/rookie-ninja/rk-grpc/v2/boot"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	boot := rkboot.NewBoot(rkboot.WithBootConfigPath("boot.yaml", nil))

	postgresEntry := rkpostgres.GetPostgresEntry("project-conn")
	postgresEntry.Bootstrap(context.Background())
	if postgresEntry == nil {
		log.Fatal("Missing 'project-conn' in boot.yaml")
	}
	db := postgresEntry.GetDB("diplomaflow")
	if db == nil {
		log.Fatal("Database 'diplomaflow' not found")
	}

	wfAddr := os.Getenv("WORKFLOW_SERVICE_ADDR")
	if wfAddr == "" {
		wfAddr = "workflow_service:8085"
	}
	wfConn, err := grpc.Dial(wfAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Workflow Service: %v", err)
	}
	wfClient := workflowv1.NewWorkflowServiceClient(wfConn)

	teamAddr := os.Getenv("TEAM_SERVICE_ADDR")
	if teamAddr == "" {
		teamAddr = "team_service:8084"
	}
	teamConn, err := grpc.Dial(teamAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Team Service: %v", err)
	}
	teamClient := teamv1.NewTeamServiceClient(teamConn)

	repo := project.NewRepository(db)
	svc := project.NewService(repo, wfClient, teamClient)
	handler := project.NewHandler(svc)

	grpcEntry := rkgrpc.GetGrpcEntry("project-service")
	grpcEntry.AddRegFuncGrpc(func(s *grpc.Server) {
		projectv1.RegisterProjectServiceServer(s, handler)
	})

	boot.Bootstrap(context.Background())
	boot.WaitForShutdownSig(context.Background())
}
