package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lucky720s/diplomaflow/internal/project"
	authv1 "github.com/lucky720s/diplomaflow/protobuf/auth/v1"
	projectv1 "github.com/lucky720s/diplomaflow/protobuf/project/v1"
	teamv1 "github.com/lucky720s/diplomaflow/protobuf/team/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/protobuf/workflow/v1"
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
		panic(fmt.Errorf("grpc entry not found"))
	}
	grpcEntry.AddRegFuncGrpc(func(server *grpc.Server) {
		authConn := dialWithRetry("auth_service:8082")
		authClient := authv1.NewAuthServiceClient(authConn)
		teamConn := dialWithRetry("team_service:8084")
		teamClient := teamv1.NewTeamServiceClient(teamConn)
		workflowConn := dialWithRetry("workflow_service:8085")
		workflowClient := workflowv1.NewWorkflowServiceClient(workflowConn)
		var repo project.Repository
		var err error
		for i := 0; i < 10; i++ {
			repo, err = project.NewRepository()
			if err == nil {
				fmt.Println("successfully created repository with db connection")
				break
			}
			fmt.Printf("Waiting for DB connection attempt %d. Error was: %v\n", i+1, err)
			time.Sleep(time.Second * 2)
		}
		if err != nil {
			panic(fmt.Errorf("failed to create repository: %v", err))
		}
		handler := project.NewHandler(repo, authClient, teamClient, workflowClient)
		projectv1.RegisterProjectServiceServer(server, handler)
	})
	boot.Bootstrap(context.Background())
	boot.WaitForShutdownSig(context.Background())
}
