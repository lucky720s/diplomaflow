package main

import (
	"context"
	"log"
	"os"

	"github.com/lucky720s/diplomaflow/internal/team"
	"github.com/lucky720s/diplomaflow/pkg/broker"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	rkboot "github.com/rookie-ninja/rk-boot/v2"
	rkpostgres "github.com/rookie-ninja/rk-db/postgres"
	rkgrpc "github.com/rookie-ninja/rk-grpc/v2/boot"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	boot := rkboot.NewBoot(rkboot.WithBootConfigPath("boot.yaml", nil))

	postgresEntry := rkpostgres.GetPostgresEntry("team-conn")
	postgresEntry.Bootstrap(context.Background())
	if postgresEntry == nil {
		log.Fatal("Missing 'team-conn' in boot.yaml")
	}
	db := postgresEntry.GetDB("diplomaflow")
	if db == nil {
		log.Fatal("Database 'diplomaflow' not found")
	}

	authAddr := os.Getenv("AUTH_SERVICE_ADDR")
	if authAddr == "" {
		authAddr = "auth_service:8082"
	}
	authConn, err := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Auth Service: %v", err)
	}
	authClient := authv1.NewAuthServiceClient(authConn)

	repo := team.NewRepository(db)
	svc := team.NewService(repo, authClient)
	kafkaAddr := os.Getenv("KAFKA_ADDR")
	if kafkaAddr == "" {
		kafkaAddr = "kafka:9092"
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	broker.StartConsumer(ctx, []string{kafkaAddr}, "team-service-group", "project-events", func(event broker.Event) error {
		if event.Type == "ProjectCreated" {
			// Парсим payload и создаем команду
			// Тут нужно привести типы map[string]interface{} к конкретным
			// svc.CreateTeam(...)
			log.Println("Received ProjectCreated event, creating team...")
		}
		return nil
	})
	handler := team.NewHandler(svc)

	grpcEntry := rkgrpc.GetGrpcEntry("team-service")
	grpcEntry.AddRegFuncGrpc(func(s *grpc.Server) {
		teamv1.RegisterTeamServiceServer(s, handler)
	})

	boot.Bootstrap(context.Background())
	boot.WaitForShutdownSig(context.Background())
}
