package notification

import (
	"context"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	notificationv1.UnimplementedNotificationServiceServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) SendNotification(ctx context.Context, req *notificationv1.SendNotificationRequest) (*notificationv1.SendNotificationResponse, error) {
	id, err := h.service.SendNotification(ctx, req.UserId, req.Title, req.Message, req.Link, req.Type)
	if err != nil {
		return nil, err
	}
	return &notificationv1.SendNotificationResponse{NotificationId: id}, nil
}

func (h *Handler) ListNotifications(ctx context.Context, req *notificationv1.ListNotificationsRequest) (*notificationv1.ListNotificationsResponse, error) {
	list, total, err := h.service.ListNotifications(ctx, req.UserId, req.OnlyUnread, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	var pbList []*notificationv1.Notification
	for _, n := range list {
		pbList = append(pbList, &notificationv1.Notification{
			Id:        n.ID,
			Title:     n.Title,
			Message:   n.Message,
			Link:      n.Link,
			Type:      n.Type,
			IsRead:    n.IsRead,
			CreatedAt: timestamppb.New(n.CreatedAt),
		})
	}

	return &notificationv1.ListNotificationsResponse{Notifications: pbList, TotalCount: total}, nil
}

func (h *Handler) MarkAsRead(ctx context.Context, req *notificationv1.MarkAsReadRequest) (*emptypb.Empty, error) {
	err := h.service.MarkAsRead(ctx, req.NotificationId, req.UserId)
	return &emptypb.Empty{}, err
}
