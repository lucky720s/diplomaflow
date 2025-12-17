package tests_notification

import (
	"context"
	"testing"

	"github.com/lucky720s/diplomaflow/internal/notification"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_SendNotification(t *testing.T) {
	repo := new(MockRepository)
	log := logger.New("test")
	svc := notification.NewService(repo, log)

	repo.On("Create", mock.Anything, mock.MatchedBy(func(n *notification.Notification) bool {
		return n.UserID == 1 && n.Title == "Hello" && n.Type == "info"
	})).Return(nil)

	id, err := svc.SendNotification(context.Background(), 1, "Hello", "Msg", "link", "info")

	require.NoError(t, err)
	require.Equal(t, int64(1), id)
	repo.AssertExpectations(t)
}

func TestHandler_SendNotification(t *testing.T) {
	mockSvc := new(MockNotificationService)
	handler := notification.NewHandler(mockSvc)

	mockSvc.On("SendNotification", mock.Anything, int64(10), "Title", "Msg", "http://link", "alert").
		Return(int64(999), nil)

	req := &notificationv1.SendNotificationRequest{
		UserId:  10,
		Title:   "Title",
		Message: "Msg",
		Link:    "http://link",
		Type:    "alert",
	}

	resp, err := handler.SendNotification(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, int64(999), resp.NotificationId)
}
