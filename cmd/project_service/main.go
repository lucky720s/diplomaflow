package main

import (
	"context"

	"github.com/lucky720s/diplomaflow/internal/project"
	project_pb "github.com/lucky720s/diplomaflow/pkg/protobuf/project"
	rkboot "github.com/rookie-ninja/rk-boot/v2"
	rkgrpc "github.com/rookie-ninja/rk-grpc/v2/boot"
	"google.golang.org/grpc"
)

func main() {
	boot := rkboot.NewBoot()
	grpcEntry := rkgrpc.GetGrpcEntry("project-service")
	grpcEntry.AddRegFuncGrpc(func(server *grpc.Server) {
		project_pb.RegisterProjectServiceServer(server, project.NewHandler())
	})
	boot.Bootstrap(context.Background())
	boot.WaitForShutdownSig(context.Background())
}
