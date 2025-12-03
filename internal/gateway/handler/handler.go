package handler

import (
	"fmt"

	"github.com/lucky720s/diplomaflow/internal/gateway/config"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	rolev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/role/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	universityv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/university/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Handler struct {
	authClient       authv1.AuthServiceClient
	projectClient    projectv1.ProjectServiceClient
	teamClient       teamv1.TeamServiceClient
	universityClient universityv1.UniversityServiceClient
	roleClient       rolev1.RoleServiceClient
	workflowClient   workflowv1.WorkflowServiceClient
}

func NewHandler(cfg *config.Config) (*Handler, error) {
	connect := func(addr string) (*grpc.ClientConn, error) {
		return grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	authConn, err := connect(cfg.AuthServiceAddr)
	if err != nil {
		return nil, fmt.Errorf("auth connect: %w", err)
	}

	projectConn, err := connect(cfg.ProjectServiceAddr)
	if err != nil {
		return nil, fmt.Errorf("project connect: %w", err)
	}

	teamConn, err := connect(cfg.TeamServiceAddr)
	if err != nil {
		return nil, fmt.Errorf("team connect: %w", err)
	}

	univConn, err := connect(cfg.UniversityServiceAddr)
	if err != nil {
		return nil, fmt.Errorf("univ connect: %w", err)
	}

	roleConn, err := connect(cfg.RoleServiceAddr)
	if err != nil {
		return nil, fmt.Errorf("role connect: %w", err)
	}

	wfConn, err := connect(cfg.WorkflowServiceAddr)
	if err != nil {
		return nil, fmt.Errorf("workflow connect: %w", err)
	}

	return &Handler{
		authClient:       authv1.NewAuthServiceClient(authConn),
		projectClient:    projectv1.NewProjectServiceClient(projectConn),
		teamClient:       teamv1.NewTeamServiceClient(teamConn),
		universityClient: universityv1.NewUniversityServiceClient(univConn),
		roleClient:       rolev1.NewRoleServiceClient(roleConn),
		workflowClient:   workflowv1.NewWorkflowServiceClient(wfConn),
	}, nil
}
