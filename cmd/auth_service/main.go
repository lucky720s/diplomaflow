package main

import (
	"context"
	"log"
	"os"

	"github.com/lucky720s/diplomaflow/internal/auth"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	rolev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/role/v1"
	universityv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/university/v1"
	rkboot "github.com/rookie-ninja/rk-boot/v2"
	rkpostgres "github.com/rookie-ninja/rk-db/postgres"
	rkgrpc "github.com/rookie-ninja/rk-grpc/v2/boot"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	boot := rkboot.NewBoot(rkboot.WithBootConfigPath("boot.yaml", nil))
	postgresEntry := rkpostgres.GetPostgresEntry("auth-conn")
	postgresEntry.Bootstrap(context.Background())
	if postgresEntry == nil {
		log.Fatal("Missing 'auth-conn' in boot.yaml ")
	}

	db := postgresEntry.GetDB("diplomaflow")
	if db == nil {
		log.Fatal("Database 'diplomaflow' not found in postgres entry")
	}

	univAddr := os.Getenv("UNIVERSITY_SERVICE_ADDR")
	if univAddr == "" {
		univAddr = "university_service:8081"
	}
	univConn, err := grpc.Dial(univAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to University Service: %v", err)
	}
	univClient := universityv1.NewUniversityServiceClient(univConn)

	roleAddr := os.Getenv("ROLE_SERVICE_ADDR")
	if roleAddr == "" {
		roleAddr = "role_service:8086"
	}
	roleConn, err := grpc.Dial(roleAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Role Service: %v", err)
	}
	roleClient := rolev1.NewRoleServiceClient(roleConn)

	repo := auth.NewRepository(db)

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "default_secret"
	}
	jwtWrapper := auth.JwtWrapper{
		SecretKey:       jwtSecret,
		Issuer:          "diplomaflow",
		ExpirationHours: 24,
	}

	svc := auth.NewService(repo, jwtWrapper, univClient, roleClient)
	handler := auth.NewHandler(svc)

	grpcEntry := rkgrpc.GetGrpcEntry("auth-service")
	grpcEntry.AddRegFuncGrpc(func(s *grpc.Server) {
		authv1.RegisterAuthServiceServer(s, handler)
	})

	boot.Bootstrap(context.Background())
	boot.WaitForShutdownSig(context.Background())
}
