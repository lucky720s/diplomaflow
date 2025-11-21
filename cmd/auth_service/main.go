package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lucky720s/diplomaflow/internal/auth"
	authv1 "github.com/lucky720s/diplomaflow/protobuf/auth/v1"
	rolev1 "github.com/lucky720s/diplomaflow/protobuf/role/v1"
	universityv1 "github.com/lucky720s/diplomaflow/protobuf/university/v1"
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
	grpcEntry := rkgrpc.GetGrpcEntry("auth-service")
	if grpcEntry == nil {
		panic(fmt.Errorf("failed to get gRPC entry: %v", grpcEntry))
	}
	grpcEntry.AddRegFuncGrpc(func(server *grpc.Server) {
		universityConn := dialWithRetry("university_service:8081")
		universityClient := universityv1.NewUniversityServiceClient(universityConn)
		roleConn := dialWithRetry("role_service:8086")
		roleClient := rolev1.NewRoleServiceClient(roleConn)

		var repo auth.Repository
		var err error
		for i := 0; i < 10; i++ {
			repo, err = auth.NewRepository()
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
		handler := auth.NewHandler(repo, universityClient, roleClient)
		authv1.RegisterAuthServiceServer(server, handler)
	})
	boot.Bootstrap(context.Background())
	boot.WaitForShutdownSig(context.Background())
}
