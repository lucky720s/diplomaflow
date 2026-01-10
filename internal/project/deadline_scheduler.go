package project

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DeadlineScheduler struct {
	repo   Repository
	logger *zap.Logger
	stopCh chan struct{}
}

func NewDeadlineScheduler(repo Repository, logger *zap.Logger) *DeadlineScheduler {
	return &DeadlineScheduler{
		repo:   repo,
		logger: logger,
		stopCh: make(chan struct{}),
	}
}

func (s *DeadlineScheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	s.logger.Info("Deadline scheduler started")
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkDeadlines(ctx)
		}
	}
}

func (s *DeadlineScheduler) checkDeadlines(ctx context.Context) {
	projects, err := s.repo.GetProjectsWithExpiredDeadlines(ctx)
	if err != nil {
		s.logger.Error("Failed to get projects with expired deadlines", zap.Error(err))
		return
	}

	for _, p := range projects {
		projectID := p.ID

		err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
			var locked Project
			if err := tx.WithContext(ctx).
				Clauses(clause.Locking{Strength: "UPDATE"}).
				First(&locked, "id = ?", projectID).Error; err != nil {
				return err
			}

			// idempotency inside lock
			if locked.Status != "active" || locked.DeadlineProcessed || locked.DeadlineAt == nil {
				return nil
			}
			if locked.DeadlineAt.After(time.Now().UTC()) {
				return nil
			}

			// mark processed
			if err := tx.WithContext(ctx).Model(&Project{}).
				Where("id = ?", locked.ID).
				Updates(map[string]interface{}{
					"deadline_processed": true,
					"updated_at":         time.Now().UTC(),
				}).Error; err != nil {
				return err
			}

			// enqueue event to workflow-actions (workflow_service will execute ON_DEADLINE)
			payload := map[string]interface{}{
				"project_id":    locked.ID,
				"state_id":      locked.CurrentStateID,
				"department_id": locked.DepartmentID,
				"trigger":       "ON_DEADLINE",
				"deadline_at":   locked.DeadlineAt.UTC().Format(time.RFC3339),
			}
			b, _ := json.Marshal(payload)

			ev := &OutboxEvent{
				Topic:         "workflow-actions",
				EventType:     "WorkflowDeadlineReached",
				AggregateType: "project",
				AggregateID:   fmt.Sprint(locked.ID),
				Payload:       datatypes.JSON(b),
				Status:        "pending",
				CreatedAt:     time.Now().UTC(),
			}
			return tx.WithContext(ctx).Create(ev).Error
		})

		if err != nil {
			s.logger.Error("Failed to process deadline", zap.Int64("project_id", projectID), zap.Error(err))
		}
	}
}

func (s *DeadlineScheduler) Stop() {
	close(s.stopCh)
}
