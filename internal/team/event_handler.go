package team

import (
	"context"
	"errors"
	"fmt"
	"strings"

	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	workflowv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/workflow/v1"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ProjectCreatedEvent struct {
	ProjectID    int64  `json:"project_id"`
	StudentID    int64  `json:"student_id"`
	DepartmentID int64  `json:"department_id"`
	FirstStateID int64  `json:"first_state_id"`
	Title        string `json:"title"`
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

func (h *EventHandler) HandleProjectCreated(ctx context.Context, event ProjectCreatedEvent) error {
	h.logger.Info("Processing ProjectCreated event", zap.Int64("project_id", event.ProjectID))
	if event.FirstStateID == 0 {
		return h.createTeamWithFallback(ctx, event)
	}
	stepConfig, err := h.workflowClient.GetStepConfiguration(ctx, &workflowv1.GetStepConfigurationRequest{
		StateId: event.FirstStateID,
	})
	if err != nil {
		h.logger.Warn("Failed to get step configuration, creating team by default", zap.Error(err))
		return h.createTeamWithFallback(ctx, event)
	}

	teamConfig := stepConfig.GetTeamConfig()
	if teamConfig == nil || teamConfig.GetAllowSolo() || teamConfig.GetMinSize() <= 1 {
		return h.createTeamWithFallback(ctx, event)
	}
	_, notifErr := h.notifClient.SendNotification(ctx, &notificationv1.SendNotificationRequest{
		UserId:  event.StudentID,
		Title:   "Соберите команду",
		Message: fmt.Sprintf("Для проекта '%s' требуется минимум %d участников", event.Title, teamConfig.GetMinSize()),
		Link:    "/teams/create",
		Type:    "TEAM_REQUIRED",
	})
	if notifErr != nil {
		h.logger.Error("Failed to send team required notification", zap.Error(notifErr))
	}

	return nil
}
func (h *EventHandler) createTeamWithFallback(ctx context.Context, event ProjectCreatedEvent) error {
	err := h.service.CreateTeamForProject(ctx, event.ProjectID, event.StudentID)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "duplicate key") {
			h.logger.Warn("Team for project already exists, skipping (idempotency)",
				zap.Int64("project_id", event.ProjectID))
			return nil
		}
		h.logger.Error("Failed to create team", zap.Error(err))
		return err
	}
	return nil
}
