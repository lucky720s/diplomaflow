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

	res, err := h.notificationClient.ListNotifications(c.Request.Context(), &notificationv1.ListNotificationsRequest{
		UserId:     userID,
		OnlyUnread: onlyUnread,
		Page:       1,
		PageSize:   20,
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
	id, _ := strconv.ParseInt(idStr, 10, 64)

	_, err := h.notificationClient.MarkAsRead(c.Request.Context(), &notificationv1.MarkAsReadRequest{
		NotificationId: id,
		UserId:         userID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
