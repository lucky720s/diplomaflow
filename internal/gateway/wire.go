//go:generate wire
//go:build wireinject
// +build wireinject

package gateway

import (
	"time"

	"github.com/google/wire"
	"github.com/lucky720s/diplomaflow/internal/gateway/config"
	"github.com/lucky720s/diplomaflow/internal/gateway/handler"
	grpcpkg "github.com/lucky720s/diplomaflow/pkg/grpc"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	chatv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/chat/v1"
	filev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/file/v1"
	formv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/form/v1"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	taskv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/task/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	universityv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/university/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func provideConn(addr string) (*grpc.ClientConn, func(), error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(grpcpkg.DefaultRetryServiceConfig()),
		grpc.WithUnaryInterceptor(grpcpkg.TimeoutInterceptor(10*time.Second)),
	)
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() { conn.Close() }
	return conn, cleanup, nil
}

func ProvideAuthClient(cfg *config.Config) (authv1.AuthServiceClient, func(), error) {
	conn, cleanup, err := provideConn(cfg.AuthServiceAddr)
	if err != nil {
		return nil, nil, err
	}
	return authv1.NewAuthServiceClient(conn), cleanup, nil
}

func ProvideProjectClient(cfg *config.Config) (projectv1.ProjectServiceClient, func(), error) {
	conn, cleanup, err := provideConn(cfg.ProjectServiceAddr)
	if err != nil {
		return nil, nil, err
	}
	return projectv1.NewProjectServiceClient(conn), cleanup, nil
}

func ProvideTeamClient(cfg *config.Config) (teamv1.TeamServiceClient, func(), error) {
	conn, cleanup, err := provideConn(cfg.TeamServiceAddr)
	if err != nil {
		return nil, nil, err
	}
	return teamv1.NewTeamServiceClient(conn), cleanup, nil
}

func ProvideUniversityClient(cfg *config.Config) (universityv1.UniversityServiceClient, func(), error) {
	conn, cleanup, err := provideConn(cfg.UniversityServiceAddr)
	if err != nil {
		return nil, nil, err
	}
	return universityv1.NewUniversityServiceClient(conn), cleanup, nil
}

func ProvideWorkflowClient(cfg *config.Config) (workflowv1.WorkflowServiceClient, func(), error) {
	conn, cleanup, err := provideConn(cfg.WorkflowServiceAddr)
	if err != nil {
		return nil, nil, err
	}
	return workflowv1.NewWorkflowServiceClient(conn), cleanup, nil
}

func ProvideNotificationClient(cfg *config.Config) (notificationv1.NotificationServiceClient, func(), error) {
	conn, cleanup, err := provideConn(cfg.NotificationServiceAddr)
	if err != nil {
		return nil, nil, err
	}
	return notificationv1.NewNotificationServiceClient(conn), cleanup, nil
}

func ProvideFileClient(cfg *config.Config) (filev1.FileServiceClient, func(), error) {
	conn, cleanup, err := provideConn(cfg.FileServiceAddr)
	if err != nil {
		return nil, nil, err
	}
	return filev1.NewFileServiceClient(conn), cleanup, nil
}

func ProvideFormClient(cfg *config.Config) (formv1.FormServiceClient, func(), error) {
	conn, cleanup, err := provideConn(cfg.FormServiceAddr)
	if err != nil {
		return nil, nil, err
	}
	return formv1.NewFormServiceClient(conn), cleanup, nil
}

func ProvideAdminClient(cfg *config.Config) (adminv1.AdminServiceClient, func(), error) {
	conn, cleanup, err := provideConn(cfg.AdminServiceAddr)
	if err != nil {
		return nil, nil, err
	}
	return adminv1.NewAdminServiceClient(conn), cleanup, nil
}

func ProvideTaskClient(cfg *config.Config) (taskv1.TaskServiceClient, func(), error) {
	conn, cleanup, err := provideConn(cfg.TaskServiceAddr)
	if err != nil {
		return nil, nil, err
	}
	return taskv1.NewTaskServiceClient(conn), cleanup, nil
}
func ProvideNormControlClient(cfg *config.Config) (adminv1.NormControlServiceClient, func(), error) {
	conn, cleanup, err := provideConn(cfg.AdminServiceAddr)
	if err != nil {
		return nil, nil, err
	}
	return adminv1.NewNormControlServiceClient(conn), cleanup, nil
}

func ProvideChatClient(cfg *config.Config) (chatv1.ChatServiceClient, func(), error) {
	conn, cleanup, err := provideConn(cfg.ChatServiceAddr)
	if err != nil {
		return nil, nil, err
	}
	return chatv1.NewChatServiceClient(conn), cleanup, nil
}

func InitializeApp(cfg *config.Config, log *logger.Logger) (*handler.Handler, func(), error) {
	wire.Build(
		ProvideAuthClient,
		ProvideProjectClient,
		ProvideTeamClient,
		ProvideUniversityClient,
		ProvideWorkflowClient,
		ProvideNotificationClient,
		ProvideFileClient,
		ProvideFormClient,
		ProvideTaskClient,
		ProvideAdminClient,
		ProvideNormControlClient,
		ProvideChatClient,
		handler.NewHandler,
	)
	return &handler.Handler{}, nil, nil
}
