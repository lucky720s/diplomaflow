package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lucky720s/diplomaflow/internal/project"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	rkboot "github.com/rookie-ninja/rk-boot/v2"
	rkgrpc "github.com/rookie-ninja/rk-grpc/v2/boot"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func dialWithRetry(target string) *grpc.ClientConn {
	var conn *grpc.ClientConn
	var err error
	for i := 0; i < 5; i++ {
		conn, err = grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			return conn
		}
		fmt.Println("Retrying connection to", target)
		time.Sleep(time.Second * 2)
	}
	panic(fmt.Errorf("failed to connect to %s: %v", target, err))
}
func main() {
	boot := rkboot.NewBoot()
	grpcEntry := rkgrpc.GetGrpcEntry("project-service")
	if grpcEntry == nil {
		panic(fmt.Errorf("grpc entry 'project-service' not found"))
	}

	grpcEntry.AddRegFuncGrpc(func(server *grpc.Server) {
		workflowSvcAddr := os.Getenv("WORKFLOW_SERVICE_ADDR")
		if workflowSvcAddr == "" {
			workflowSvcAddr = "workflow_service:8085"
		}
		conn := dialWithRetry(workflowSvcAddr)
		wfClient := workflowv1.NewWorkflowServiceClient(conn)
		log.Println("Connected to workflow_service.")
		repo, err := project.NewRepository(wfClient)
		if err != nil {
			panic(fmt.Errorf("failed to create repository for project_service: %v", err))
		}
		log.Println("Repository for project_service created successfully.")
		handler := project.NewHandler(repo)
		projectv1.RegisterProjectServiceServer(server, handler)
	})

	boot.Bootstrap(context.Background())
	boot.WaitForShutdownSig(context.Background())
}
