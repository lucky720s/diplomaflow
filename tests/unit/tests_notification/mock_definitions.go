package tests_notification

import (
	"context"

	"github.com/lucky720s/diplomaflow/internal/notification"
	"github.com/stretchr/testify/mock"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, n *notification.Notification) error {
	args := m.Called(ctx, n)
	if n.ID == 0 {
		n.ID = 1
	}
	return args.Error(0)
}

func (m *MockRepository) List(ctx context.Context, userID int64, onlyUnread bool, limit, offset int) ([]*notification.Notification, int64, error) {
	args := m.Called(ctx, userID, onlyUnread, limit, offset)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*notification.Notification), args.Get(1).(int64), args.Error(2)
}

func (m *MockRepository) MarkAsRead(ctx context.Context, id int64, userID int64) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

type MockNotificationService struct {
	mock.Mock
}

func (m *MockNotificationService) SendNotification(ctx context.Context, userID int64, title, message, link, nType string) (int64, error) {
	args := m.Called(ctx, userID, title, message, link, nType)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockNotificationService) ListNotifications(ctx context.Context, userID int64, onlyUnread bool, page, pageSize int32) ([]*notification.Notification, int64, error) {
	args := m.Called(ctx, userID, onlyUnread, page, pageSize)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*notification.Notification), args.Get(1).(int64), args.Error(2)
}

func (m *MockNotificationService) MarkAsRead(ctx context.Context, id, userID int64) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}
