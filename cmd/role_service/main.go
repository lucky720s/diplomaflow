package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lucky720s/diplomaflow/internal/role"
	rolev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/role/v1"
	rkboot "github.com/rookie-ninja/rk-boot/v2"
	rkgrpc "github.com/rookie-ninja/rk-grpc/v2/boot"
	"google.golang.org/grpc"
)

func main() {
	boot := rkboot.NewBoot()
	grpcEntry := rkgrpc.GetGrpcEntry("role-service")
	if grpcEntry == nil {
		panic(fmt.Errorf("grpc entry 'role-service' not found in boot.yaml"))
	}
	grpcEntry.AddRegFuncGrpc(func(server *grpc.Server) {
		var repo role.Repository
		var err error
		for i := 0; i < 10; i++ {
			repo, err = role.NewRepository()
			if err == nil {
				fmt.Println("Successfully created repository with DB connection")
				break
			}
			fmt.Printf("Waiting for DB connection to be ready... attempt %d. Error: %v\n", i+1, err)
			time.Sleep(2 * time.Second)
		}
		if err != nil {
			panic(fmt.Errorf("failed to create repository after multiple attempts: %w", err))
		}
		handler := role.NewHandler(repo)
		rolev1.RegisterRoleServiceServer(server, handler)
	})
	boot.Bootstrap(context.Background())
	boot.WaitForShutdownSig(context.Background())
}
