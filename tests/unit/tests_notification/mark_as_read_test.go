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

func TestService_MarkAsRead(t *testing.T) {
	repo := new(MockRepository)
	log := logger.New("test")
	svc := notification.NewService(repo, log)

	repo.On("MarkAsRead", mock.Anything, int64(100), int64(5)).
		Return(nil)

	err := svc.MarkAsRead(context.Background(), 100, 5)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestHandler_MarkAsRead(t *testing.T) {
	mockSvc := new(MockNotificationService)
	handler := notification.NewHandler(mockSvc)

	mockSvc.On("MarkAsRead", mock.Anything, int64(123), int64(456)).
		Return(nil)

	req := &notificationv1.MarkAsReadRequest{
		NotificationId: 123,
		UserId:         456,
	}

	resp, err := handler.MarkAsRead(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, resp)
}
