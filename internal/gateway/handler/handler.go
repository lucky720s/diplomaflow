package handler

import (
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	rolev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/role/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	universityv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/university/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
)

type Handler struct {
	authClient       authv1.AuthServiceClient
	projectClient    projectv1.ProjectServiceClient
	teamClient       teamv1.TeamServiceClient
	universityClient universityv1.UniversityServiceClient
	roleClient       rolev1.RoleServiceClient
	workflowClient   workflowv1.WorkflowServiceClient
}

func NewHandler(
	authClient authv1.AuthServiceClient,
	projectClient projectv1.ProjectServiceClient,
	teamClient teamv1.TeamServiceClient,
	universityClient universityv1.UniversityServiceClient,
	roleClient rolev1.RoleServiceClient,
	workflowClient workflowv1.WorkflowServiceClient,
) *Handler {
	return &Handler{
		authClient:       authClient,
		projectClient:    projectClient,
		teamClient:       teamClient,
		universityClient: universityClient,
		roleClient:       roleClient,
		workflowClient:   workflowClient,
	}
}
