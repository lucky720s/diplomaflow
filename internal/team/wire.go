//go:generate wire
//go:build wireinject
// +build wireinject

package team

import (
	"strings"

	"github.com/google/wire"
	"github.com/lucky720s/diplomaflow/pkg/broker"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
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

func ProvideAuthClient(cfg *Config) (authv1.AuthServiceClient, func(), error) {
	conn, err := grpc.NewClient(cfg.Services.AuthAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { conn.Close() }
	return authv1.NewAuthServiceClient(conn), cleanup, nil
}

func ProvideConsumer(cfg *Config, log *logger.Logger) (*broker.Consumer, func(), error) {
	brokers := strings.Split(cfg.Kafka.Brokers, ",")
	consumer, err := broker.NewConsumer(brokers, cfg.Kafka.GroupID, log.Logger)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { consumer.Close() }
	return consumer, cleanup, nil
}

type AppContainer struct {
	Handler      *Handler
	Consumer     *broker.Consumer
	EventHandler *EventHandler
}

func InitializeApp(cfg *Config, log *logger.Logger) (*AppContainer, func(), error) {
	wire.Build(
		ProvideDB,
		ProvideAuthClient,
		ProvideConsumer,
		NewRepository,
		NewService,
		NewEventHandler,
		NewHandler,
		wire.Struct(new(AppContainer), "*"),
	)
	return &AppContainer{}, nil, nil
}
