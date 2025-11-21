package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lucky720s/diplomaflow/internal/team"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	rkboot "github.com/rookie-ninja/rk-boot/v2"
	rkgrpc "github.com/rookie-ninja/rk-grpc/v2/boot"
	"google.golang.org/grpc"
)

func main() {
	boot := rkboot.NewBoot()
	grpcEntry := rkgrpc.GetGrpcEntry("team-service")
	if grpcEntry == nil {
		panic(fmt.Errorf("failed to get gRPC entry"))
	}
	grpcEntry.AddRegFuncGrpc(func(server *grpc.Server) {
		var repo team.Repository
		var err error
		for i := 0; i < 10; i++ {
			repo, err = team.NewRepository()
			if err == nil {
				fmt.Println("team connect success")
				break
			}
			fmt.Printf("Waiting for DB connect atetemp %d. Error %v\n", i+1, err)
			time.Sleep(2 * time.Second)
		}
		if err != nil {
			panic(fmt.Errorf("failed to init repo: %v", err))
		}
		handler := team.NewHandler(repo)
		teamv1.RegisterTeamServiceServer(server, handler)
	})
	boot.Bootstrap(context.Background())
	boot.WaitForShutdownSig(context.Background())
}
