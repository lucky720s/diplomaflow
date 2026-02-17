package handler

import (
	adminv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/admin/v1"
	authv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/auth/v1"
	filev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/file/v1"
	formv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/form/v1"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	projectv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/project/v1"
	taskv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/task/v1"
	teamv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/team/v1"
	universityv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/university/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
)

type Handler struct {
	authClient         authv1.AuthServiceClient
	projectClient      projectv1.ProjectServiceClient
	teamClient         teamv1.TeamServiceClient
	universityClient   universityv1.UniversityServiceClient
	workflowClient     workflowv1.WorkflowServiceClient
	notificationClient notificationv1.NotificationServiceClient
	fileClient         filev1.FileServiceClient
	formClient         formv1.FormServiceClient
	adminClient        adminv1.AdminServiceClient
	taskClient         taskv1.TaskServiceClient
}

func NewHandler(
	authClient authv1.AuthServiceClient,
	projectClient projectv1.ProjectServiceClient,
	teamClient teamv1.TeamServiceClient,
	universityClient universityv1.UniversityServiceClient,
	workflowClient workflowv1.WorkflowServiceClient,
	notificationClient notificationv1.NotificationServiceClient,
	fileClient filev1.FileServiceClient,
	formClient formv1.FormServiceClient,
	adminClient adminv1.AdminServiceClient,
	taskClient taskv1.TaskServiceClient,
) *Handler {
	return &Handler{
		authClient:         authClient,
		projectClient:      projectClient,
		teamClient:         teamClient,
		universityClient:   universityClient,
		workflowClient:     workflowClient,
		notificationClient: notificationClient,
		fileClient:         fileClient,
		formClient:         formClient,
		adminClient:        adminClient,
		taskClient:         taskClient,
	}
}
