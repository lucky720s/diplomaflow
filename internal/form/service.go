package form

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

type Service struct {
	repo   Repository
	logger *logger.Logger
}

func NewService(repo Repository, log *logger.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: log,
	}
}

func (s *Service) SubmitForm(ctx context.Context, projectID, stepID, userID int64, data map[string]interface{}) (string, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		s.logger.Error("failed to marshal form data", zap.Error(err))
		return "", err
	}

	submission := &FormSubmission{
		ID:        uuid.New().String(),
		ProjectID: projectID,
		StepID:    stepID,
		UserID:    userID,
		Data:      datatypes.JSON(dataBytes),
		CreatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, submission); err != nil {
		s.logger.Error("failed to create form submission", zap.Error(err))
		return "", err
	}

	s.logger.Info("Form submitted", zap.String("id", submission.ID), zap.Int64("project_id", projectID))
	return submission.ID, nil
}

func (s *Service) GetFormSubmission(ctx context.Context, id string) (*FormSubmission, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListProjectForms(ctx context.Context, projectID int64) ([]*FormSubmission, error) {
	return s.repo.ListByProject(ctx, projectID)
}
