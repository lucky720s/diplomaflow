package project

import (
	"context"
	"strconv"
	"time"

	"go.uber.org/zap"
)

type DeadlineScheduler struct {
	repo           Repository
	actionExecutor *StateActionExecutor
	logger         *zap.Logger
	stopCh         chan struct{}
}

func NewDeadlineScheduler(repo Repository, executor *StateActionExecutor, logger *zap.Logger) *DeadlineScheduler {
	return &DeadlineScheduler{
		repo:           repo,
		actionExecutor: executor,
		logger:         logger,
		stopCh:         make(chan struct{}),
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

	for _, project := range projects {
		stateID, _ := strconv.ParseInt(project.CurrentStepID, 10, 64)

		if err := s.actionExecutor.ExecuteActions(ctx, stateID, "ON_DEADLINE", project); err != nil {
			s.logger.Error("Failed to execute ON_DEADLINE actions",
				zap.Uint("project_id", project.ID),
				zap.Error(err))
		}
	}
}

func (s *DeadlineScheduler) Stop() {
	close(s.stopCh)
}
