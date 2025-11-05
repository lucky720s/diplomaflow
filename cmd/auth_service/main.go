package main

import (
	"context"

	"github.com/lucky720s/diplomaflow/internal/auth"
	auth_pb "github.com/lucky720s/diplomaflow/pkg/protobuf/auth"
	rkboot "github.com/rookie-ninja/rk-boot/v2"
	rkgrpc "github.com/rookie-ninja/rk-grpc/v2/boot"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	boot := rkboot.NewBoot()
	grpcEntry := rkgrpc.GetGrpcEntry("auth-service")
	grpcEntry.AddRegFuncGrpc(func(server *grpc.Server) {
		auth_pb.RegisterAuthServiceServer(server, auth.NewHandler())
		reflection.Register(server)
	})
	boot.Bootstrap(context.Background())
	boot.WaitForShutdownSig(context.Background())
}
