package team

import (
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
)

type ProjectCreatedEvent struct {
	ProjectID      int64  `json:"project_id"`
	StudentID      int64  `json:"student_id"`
	DepartmentID   int64  `json:"department_id"`
	TeamID         int64  `json:"team_id"`
	FirstStateID   int64  `json:"first_state_id"`
	InitialStateID int64  `json:"initial_state_id"`
	Title          string `json:"title"`
}

type EventHandler struct {
	service        *Service
	workflowClient workflowv1.WorkflowServiceClient
	notifClient    notificationv1.NotificationServiceClient
	logger         *zap.Logger
}

func NewEventHandler(
	service *Service,
	workflowClient workflowv1.WorkflowServiceClient,
	notifClient notificationv1.NotificationServiceClient,
	logger *zap.Logger,
) *EventHandler {
	return &EventHandler{
		service:        service,
		workflowClient: workflowClient,
		notifClient:    notifClient,
		logger:         logger,
	}
}
