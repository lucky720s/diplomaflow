package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lucky720s/diplomaflow/internal/workflow"
	workflowv1 "github.com/lucky720s/diplomaflow/protobuf/workflow/v1"
	rkboot "github.com/rookie-ninja/rk-boot/v2"
	rkgrpc "github.com/rookie-ninja/rk-grpc/v2/boot"
	"google.golang.org/grpc"
)

func main() {
	boot := rkboot.NewBoot()
	grpcEntry := rkgrpc.GetGrpcEntry("workflow-service")
	if grpcEntry == nil {
		panic(fmt.Errorf("failed to get gRPC entry"))
	}
	grpcEntry.AddRegFuncGrpc(func(server *grpc.Server) {
		var repo workflow.Repository
		var err error
		for i := 0; i < 10; i++ {
			repo, err = workflow.NewRepository()
			if err == nil {
				fmt.Println("success connect to db")
				break
			}
			fmt.Printf("waiting to connect db attmep %d. Error %v\n", i+1, err)
			time.Sleep(2 * time.Second)
		}
		if err != nil {
			panic(fmt.Errorf("failed to init repo: %v", err))
		}
		handler := workflow.NewHandler(repo)
		workflowv1.RegisterWorkflowServiceServer(server, handler)
	})
	boot.Bootstrap(context.Background())
	boot.WaitForShutdownSig(context.Background())
}
