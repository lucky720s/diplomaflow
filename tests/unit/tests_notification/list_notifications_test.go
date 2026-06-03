package tests_notification

import (
	"context"
	"testing"
	"time"

	"github.com/lucky720s/diplomaflow/internal/notification"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	"github.com/lucky720s/diplomaflow/pkg/realtime"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestService_ListNotifications(t *testing.T) {
	repo := new(MockRepository)
	log := logger.New("test")
	pub, _, _ := realtime.NewPublisher("")
	svc := notification.NewService(repo, notification.NewPusher(&notification.Config{}, log), pub, log)

	expectedList := []*notification.Notification{
		{ID: 1, Title: "Test 1"},
		{ID: 2, Title: "Test 2"},
	}

	repo.On("List", mock.Anything, int64(5), false, 10, 0).
		Return(expectedList, int64(2), nil)

	list, total, err := svc.ListNotifications(context.Background(), 5, false, 1, 10)

	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, list, 2)
}

func TestHandler_ListNotifications(t *testing.T) {
	mockSvc := new(MockNotificationService)
	handler := notification.NewHandler(mockSvc)

	now := time.Now()
	svcList := []*notification.Notification{
		{ID: 100, Title: "Alert", CreatedAt: now},
	}

	mockSvc.On("ListNotifications", mock.Anything, int64(7), true, int32(1), int32(20)).
		Return(svcList, int64(50), nil)

	req := &notificationv1.ListNotificationsRequest{
		UserId:     7,
		OnlyUnread: true,
		Page:       1,
		PageSize:   20,
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", "7"))
	resp, err := handler.ListNotifications(ctx, req)

	require.NoError(t, err)
	require.Equal(t, int64(50), resp.TotalCount)
	require.Len(t, resp.Notifications, 1)
	require.Equal(t, "Alert", resp.Notifications[0].Title)
	require.NotNil(t, resp.Notifications[0].CreatedAt)
}
