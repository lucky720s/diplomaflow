//go:generate wire
//go:build wireinject
// +build wireinject

package admin

import (
	"github.com/google/wire"
	"github.com/lucky720s/diplomaflow/pkg/database"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	filev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/file/v1"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"
)

func ProvideDB(cfg *Config) (*gorm.DB, func(), error) {
	return database.NewConnection(cfg.Database.DSN)
}

func ProvideAuthClient(cfg *Config) (authv1.AuthServiceClient, func(), error) {
	conn, err := grpc.NewClient(cfg.Services.AuthAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { conn.Close() }
	return authv1.NewAuthServiceClient(conn), cleanup, nil
}

func ProvideProjectClient(cfg *Config) (projectv1.ProjectServiceClient, func(), error) {
	conn, err := grpc.NewClient(cfg.Services.ProjectAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { conn.Close() }
	return projectv1.NewProjectServiceClient(conn), cleanup, nil
}

func ProvideTeamClient(cfg *Config) (teamv1.TeamServiceClient, func(), error) {
	conn, err := grpc.NewClient(cfg.Services.TeamAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { conn.Close() }
	return teamv1.NewTeamServiceClient(conn), cleanup, nil
}

func ProvideWorkflowClient(cfg *Config) (workflowv1.WorkflowServiceClient, func(), error) {
	conn, err := grpc.NewClient(cfg.Services.WorkflowAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { conn.Close() }
	return workflowv1.NewWorkflowServiceClient(conn), cleanup, nil
}

func ProvideNotificationClient(cfg *Config) (notificationv1.NotificationServiceClient, func(), error) {
	conn, err := grpc.NewClient(cfg.Services.NotificationAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { conn.Close() }
	return notificationv1.NewNotificationServiceClient(conn), cleanup, nil
}

func ProvideFileClient(cfg *Config) (filev1.FileServiceClient, func(), error) {
	conn, err := grpc.NewClient(cfg.Services.FileAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { conn.Close() }
	return filev1.NewFileServiceClient(conn), cleanup, nil
}

func InitializeApp(cfg *Config, log *zap.Logger) (*Handler, func(), error) {
	wire.Build(
		ProvideDB,
		ProvideAuthClient,
		ProvideProjectClient,
		ProvideTeamClient,
		ProvideWorkflowClient,
		ProvideNotificationClient,
		ProvideFileClient,
		NewRepository,
		NewService,
		NewHandler,
	)
	return &Handler{}, nil, nil
}
