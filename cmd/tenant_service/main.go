package main

import (
	"context"

	"github.com/lucky720s/diplomaflow/internal/tenant"
	tenant_pb "github.com/lucky720s/diplomaflow/pkg/protobuf/tenant"
	rkboot "github.com/rookie-ninja/rk-boot/v2"
	rkgrpc "github.com/rookie-ninja/rk-grpc/v2/boot"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	boot := rkboot.NewBoot()
	grpcEntry := rkgrpc.GetGrpcEntry("tenant-service")
	grpcEntry.AddRegFuncGrpc(func(server *grpc.Server) {
		tenant_pb.RegisterTenantServiceServer(server, tenant.NewHandler())
		reflection.Register(server)
	})
	boot.Bootstrap(context.Background())
	boot.WaitForShutdownSig(context.Background())
}
