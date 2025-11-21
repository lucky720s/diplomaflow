package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lucky720s/diplomaflow/internal/university"
	universityv1 "github.com/lucky720s/diplomaflow/protobuf/university/v1"
	rkboot "github.com/rookie-ninja/rk-boot/v2"
	rkgrpc "github.com/rookie-ninja/rk-grpc/v2/boot"
	"google.golang.org/grpc"
)

func main() {
	boot := rkboot.NewBoot()
	grpcEntry := rkgrpc.GetGrpcEntry("university-service")
	if grpcEntry == nil {
		panic(fmt.Errorf("could not find university grpc entry"))
	}
	grpcEntry.AddRegFuncGrpc(func(server *grpc.Server) {
		var repo university.Repository
		var err error
		for i := 0; i < 10; i++ {
			repo, err = university.NewRepository()
			if err == nil {
				fmt.Println("Successfully connect to db")
				break
			}
			fmt.Printf("Waiting for DB connection attempt %d. Error %v\n", i+1, err)
			time.Sleep(2 * time.Second)
		}
		if err != nil {
			panic(fmt.Errorf("could not connect to db: %v", err))
		}
		handler := university.NewHandler(repo)
		universityv1.RegisterUniversityServiceServer(server, handler)
	})
	boot.Bootstrap(context.Background())
	boot.WaitForShutdownSig(context.Background())
}
