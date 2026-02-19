package notification

import (
	"context"
	"errors"
	"time"

	"github.com/lucky720s/diplomaflow/pkg/logger"
	"go.uber.org/zap"
)

type Service struct {
	repo   Repository
	logger *logger.Logger
}

func NewService(repo Repository, log *logger.Logger) *Service {
	return &Service{repo: repo, logger: log}
}

func (s *Service) SendNotification(ctx context.Context, userID int64, title, message, link, nType string) (int64, error) {
	n := &Notification{
		UserID:    userID,
		Title:     title,
		Message:   message,
		Link:      link,
		Type:      nType,
		IsRead:    false,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, n); err != nil {
		return 0, err
	}
	return n.ID, nil
}

func (s *Service) ListNotifications(ctx context.Context, userID int64, onlyUnread bool, page, pageSize int32) ([]*Notification, int64, error) {
	if pageSize <= 0 {
		pageSize = 10
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize

	return s.repo.List(ctx, userID, onlyUnread, int(pageSize), int(offset))
}

func (s *Service) MarkAsRead(ctx context.Context, id, userID int64) error {
	return s.repo.MarkAsRead(ctx, id, userID)
}

func (s *Service) DeleteNotification(ctx context.Context, id, userID int64) error {
	return s.repo.Delete(ctx, id, userID)
}
func (s *Service) MarkAllAsRead(ctx context.Context, userID int64) (int64, error) {
	if userID <= 0 {
		return 0, errors.New("user_id is required")
	}
	count, err := s.repo.MarkAllAsRead(ctx, userID)
	if err != nil {
		s.logger.Error("MarkAllAsRead failed", zap.Int64("user_id", userID), zap.Error(err))
		return 0, err
	}
	s.logger.Info("All notifications marked as read", zap.Int64("user_id", userID), zap.Int64("count", count))
	return count, nil
}
