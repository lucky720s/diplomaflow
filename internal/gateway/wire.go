//go:generate wire
//go:build wireinject
// +build wireinject

package gateway

import (
	"github.com/google/wire"
	"github.com/lucky720s/diplomaflow/internal/gateway/config"
	"github.com/lucky720s/diplomaflow/internal/gateway/handler"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	rolev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/role/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	universityv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/university/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func provideConn(addr string) (*grpc.ClientConn, func(), error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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

func ProvideRoleClient(cfg *config.Config) (rolev1.RoleServiceClient, func(), error) {
	conn, cleanup, err := provideConn(cfg.RoleServiceAddr)
	if err != nil {
		return nil, nil, err
	}
	return rolev1.NewRoleServiceClient(conn), cleanup, nil
}

func ProvideWorkflowClient(cfg *config.Config) (workflowv1.WorkflowServiceClient, func(), error) {
	conn, cleanup, err := provideConn(cfg.WorkflowServiceAddr)
	if err != nil {
		return nil, nil, err
	}
	return workflowv1.NewWorkflowServiceClient(conn), cleanup, nil
}

func InitializeApp(cfg *config.Config, log *logger.Logger) (*handler.Handler, func(), error) {
	wire.Build(
		ProvideAuthClient,
		ProvideProjectClient,
		ProvideTeamClient,
		ProvideUniversityClient,
		ProvideRoleClient,
		ProvideWorkflowClient,
		handler.NewHandler,
	)
	return &handler.Handler{}, nil, nil
}
