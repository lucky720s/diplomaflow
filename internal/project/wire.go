//go:generate wire
//go:build wireinject
// +build wireinject

package project

import (
	"strings"

	"github.com/google/wire"
	"github.com/lucky720s/diplomaflow/pkg/broker"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ProvideDB(cfg *Config) (*gorm.DB, func(), error) {
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}
	return db, cleanup, nil
}

func ProvideWorkflowClient(cfg *Config) (workflowv1.WorkflowServiceClient, func(), error) {
	conn, err := grpc.NewClient(cfg.Services.WorkflowAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { conn.Close() }
	return workflowv1.NewWorkflowServiceClient(conn), cleanup, nil
}

func ProvideProducer(cfg *Config) (*broker.Producer, func(), error) {
	producer, err := broker.NewProducer(strings.Split(cfg.Kafka.Brokers, ","))
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { producer.Close() }
	return producer, cleanup, nil
}

func InitializeApp(cfg *Config, log *logger.Logger) (*Handler, func(), error) {
	wire.Build(
		ProvideDB,
		ProvideWorkflowClient,
		ProvideProducer,
		NewRepository,
		NewService,
		NewHandler,
	)
	return &Handler{}, nil, nil
}
