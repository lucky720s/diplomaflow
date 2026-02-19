package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
)

func (h *Handler) ListNotifications(c *gin.Context) {
	userID := c.GetInt64("userId")
	onlyUnread := c.Query("unread") == "true"

	page := int32(1)
	pageSize := int32(20)
	if p := c.Query("page"); p != "" {
		if v, err := strconv.ParseInt(p, 10, 32); err == nil && v > 0 {
			page = int32(v)
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if v, err := strconv.ParseInt(ps, 10, 32); err == nil && v > 0 {
			pageSize = int32(v)
		}
	}

	res, err := h.notificationClient.ListNotifications(outgoingCtx(c), &notificationv1.ListNotificationsRequest{
		UserId:     userID,
		OnlyUnread: onlyUnread,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) MarkNotificationRead(c *gin.Context) {
	userID := c.GetInt64("userId")
	idStr := c.Param("id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification id"})
		return
	}

	_, err = h.notificationClient.MarkAsRead(outgoingCtx(c), &notificationv1.MarkAsReadRequest{
		NotificationId: id,
		UserId:         userID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handler) DeleteNotification(c *gin.Context) {
	userID := c.GetInt64("userId")
	idStr := c.Param("id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notification id"})
		return
	}

	_, err = h.notificationClient.DeleteNotification(outgoingCtx(c), &notificationv1.DeleteNotificationRequest{
		NotificationId: id,
		UserId:         userID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
func (h *Handler) MarkAllNotificationsRead(c *gin.Context) {
	userID := c.GetInt64("userId")

	resp, err := h.notificationClient.MarkAllAsRead(
		outgoingCtx(c),
		&notificationv1.MarkAllAsReadRequest{
			UserId: userID,
		},
	)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"updated_count": resp.UpdatedCount,
	})
}
