package team

import (
	"context"
	"errors"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ProjectCreatedEvent struct {
	ProjectID int64  `json:"project_id"`
	StudentID int64  `json:"student_id"`
	Title     string `json:"title"`
}

type EventHandler struct {
	service *Service
	logger  *zap.Logger
}

func NewEventHandler(service *Service, logger *zap.Logger) *EventHandler {
	return &EventHandler{service: service, logger: logger}
}

func (h *EventHandler) HandleProjectCreated(ctx context.Context, event ProjectCreatedEvent) error {
	h.logger.Info("Processing ProjectCreated event", zap.Int64("project_id", event.ProjectID))

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
