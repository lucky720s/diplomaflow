package notification

import (
	"context"
	"strconv"
	"strings"

	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type NotificationUseCase interface {
	SendNotification(ctx context.Context, userID int64, title, message, link, nType string) (int64, error)
	ListNotifications(ctx context.Context, userID int64, onlyUnread bool, page, pageSize int32) ([]*Notification, int64, error)
	MarkAsRead(ctx context.Context, id, userID int64) error
	DeleteNotification(ctx context.Context, id, userID int64) error
}

type Handler struct {
	notificationv1.UnimplementedNotificationServiceServer
	service NotificationUseCase
}

func NewHandler(service NotificationUseCase) *Handler {
	return &Handler{service: service}
}

func userIDFromMD(ctx context.Context) (int64, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, false
	}
	vals := md.Get("x-user-id")
	if len(vals) == 0 || strings.TrimSpace(vals[0]) == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(vals[0], 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func internalServiceFromMD(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	vals := md.Get("x-internal-service")
	if len(vals) == 0 {
		return ""
	}
	return strings.TrimSpace(vals[0])
}

func isAllowedInternalSender(svc string) bool {
	switch svc {
	case "workflow_service", "admin_service", "api_gateway":
		return true
	default:
		return false
	}
}

// ==================== RPCs ====================

func (h *Handler) SendNotification(ctx context.Context, req *notificationv1.SendNotificationRequest) (*notificationv1.SendNotificationResponse, error) {
	// Internal-only, иначе это станет публичной рассылкой
	internal := internalServiceFromMD(ctx)
	if !isAllowedInternalSender(internal) {
		return nil, status.Error(codes.PermissionDenied, "SendNotification is internal-only")
	}

	if req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if strings.TrimSpace(req.Title) == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}

	id, err := h.service.SendNotification(ctx, req.UserId, req.Title, req.Message, req.Link, req.Type)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "send notification failed: %v", err)
	}
	return &notificationv1.SendNotificationResponse{NotificationId: id}, nil
}

func (h *Handler) ListNotifications(ctx context.Context, req *notificationv1.ListNotificationsRequest) (*notificationv1.ListNotificationsResponse, error) {
	authUserID, ok := userIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing x-user-id")
	}

	// Доп. защита: если клиент всё же прислал user_id и он не совпал — запрещаем
	if req.UserId != 0 && req.UserId != authUserID {
		return nil, status.Error(codes.PermissionDenied, "cannot list notifications of another user")
	}

	list, total, err := h.service.ListNotifications(ctx, authUserID, req.OnlyUnread, req.Page, req.PageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list notifications failed: %v", err)
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
	authUserID, ok := userIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing x-user-id")
	}
	if req.UserId != 0 && req.UserId != authUserID {
		return nil, status.Error(codes.PermissionDenied, "cannot modify notifications of another user")
	}
	if req.NotificationId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "notification_id is required")
	}

	err := h.service.MarkAsRead(ctx, req.NotificationId, authUserID)
	if err == ErrNotificationNotFound {
		return nil, status.Error(codes.NotFound, "notification not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "mark as read failed: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) DeleteNotification(ctx context.Context, req *notificationv1.DeleteNotificationRequest) (*emptypb.Empty, error) {
	authUserID, ok := userIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing x-user-id")
	}
	if req.UserId != 0 && req.UserId != authUserID {
		return nil, status.Error(codes.PermissionDenied, "cannot delete notifications of another user")
	}
	if req.NotificationId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "notification_id is required")
	}

	err := h.service.DeleteNotification(ctx, req.NotificationId, authUserID)
	if err == ErrNotificationNotFound {
		return nil, status.Error(codes.NotFound, "notification not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete notification failed: %v", err)
	}

	return &emptypb.Empty{}, nil
}
