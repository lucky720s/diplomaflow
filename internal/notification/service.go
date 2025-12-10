package notification

import (
	"context"
	"time"

	"github.com/lucky720s/diplomaflow/pkg/logger"
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
		CreatedAt: time.Now(),
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
