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
	pusher Pusher
	logger *logger.Logger
}

func NewService(repo Repository, pusher Pusher, log *logger.Logger) *Service {
	return &Service{repo: repo, pusher: pusher, logger: log}
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

	// Push на устройства пользователя — best-effort: ошибка доставки не должна
	// ломать создание уведомления в БД.
	s.pushToUser(ctx, userID, title, message, link)

	return n.ID, nil
}

// pushToUser отправляет push на все устройства пользователя (best-effort).
func (s *Service) pushToUser(ctx context.Context, userID int64, title, message, link string) {
	devices, err := s.repo.ListDevices(ctx, userID)
	if err != nil {
		s.logger.Warn("list devices for push failed", zap.Int64("user_id", userID), zap.Error(err))
		return
	}
	if len(devices) == 0 {
		return
	}
	tokens := make([]string, 0, len(devices))
	for _, d := range devices {
		tokens = append(tokens, d.Token)
	}
	if err := s.pusher.Push(ctx, tokens, title, message, link); err != nil {
		s.logger.Warn("push failed", zap.Int64("user_id", userID), zap.Error(err))
	}
}

func (s *Service) RegisterDevice(ctx context.Context, userID int64, token, platform string) (*DeviceToken, error) {
	if userID <= 0 {
		return nil, errors.New("user_id is required")
	}
	if token == "" {
		return nil, errors.New("token is required")
	}
	if platform == "" {
		platform = "android"
	}
	d := &DeviceToken{UserID: userID, Token: token, Platform: platform}
	if err := s.repo.UpsertDevice(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Service) UnregisterDevice(ctx context.Context, userID int64, token string) error {
	if userID <= 0 {
		return errors.New("user_id is required")
	}
	if token == "" {
		return errors.New("token is required")
	}
	return s.repo.DeleteDevice(ctx, userID, token)
}

func (s *Service) ListDevices(ctx context.Context, userID int64) ([]*DeviceToken, error) {
	if userID <= 0 {
		return nil, errors.New("user_id is required")
	}
	return s.repo.ListDevices(ctx, userID)
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
