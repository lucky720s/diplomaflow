package project

import (
	"context"
	"encoding/json"
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
	return &DeadlineScheduler{repo: repo, logger: logger, stopCh: make(chan struct{})}
}

func (s *DeadlineScheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.check(ctx)
		}
	}
}

func (s *DeadlineScheduler) check(ctx context.Context) {
	list, err := s.repo.GetProjectsWithExpiredDeadlines(ctx)
	if err != nil {
		s.logger.Error("get expired deadlines failed", zap.Error(err))
		return
	}

	for _, p := range list {
		projectID := p.ID
		err := s.repo.Transaction(ctx, func(tx *gorm.DB) error {
			var locked Project
			if err := tx.WithContext(ctx).
				Clauses(clause.Locking{Strength: "UPDATE"}).
				First(&locked, "id = ?", projectID).Error; err != nil {
				return err
			}
			if locked.Status != "active" || locked.DeadlineProcessed || locked.DeadlineAt == nil {
				return nil
			}
			if locked.DeadlineAt.After(time.Now().UTC()) {
				return nil
			}

			if err := tx.WithContext(ctx).Model(&Project{}).
				Where("id = ?", locked.ID).
				Updates(map[string]interface{}{
					"deadline_processed": true,
					"updated_at":         time.Now().UTC(),
				}).Error; err != nil {
				return err
			}

			// outbox event for workflow-actions consumer (next step)
			payload := map[string]interface{}{
				"project_id":    locked.ID,
				"state_id":      locked.CurrentStateID,
				"department_id": locked.DepartmentID,
				"trigger":       "ON_DEADLINE",
				"deadline_at":   locked.DeadlineAt.UTC().Format(time.RFC3339),
			}
			b, _ := json.Marshal(payload)

			ev := &OutboxEvent{
				Topic:     "workflow-actions",
				EventType: "WorkflowDeadlineReached",
				Payload:   datatypes.JSON(b),
				Status:    "pending",
				CreatedAt: time.Now().UTC(),
			}
			return tx.WithContext(ctx).Create(ev).Error
		})
		if err != nil {
			s.logger.Error("deadline processing failed", zap.Int64("project_id", projectID), zap.Error(err))
		}
	}
}

func (s *DeadlineScheduler) Stop() { close(s.stopCh) }
