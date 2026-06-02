package handler

// Mobile BFF (Backend-for-Frontend) handlers.
//
// Назначение: компактные, агрегированные эндпоинты под Flutter-приложение.
// Идея — отдавать мобильному клиенту данные нескольких доменов одним
// запросом, чтобы экономить раунд-трипы на медленных мобильных сетях.
// Новых доменных сервисов не вводим: переиспользуем уже подключённые
// gRPC-клиенты (как это сделано в onboarding.go).

import (
	"net/http"

	"github.com/gin-gonic/gin"
	notificationv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/notification/v1"
	taskv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/task/v1"
)

// mobileRecentLimit — сколько элементов класть в "ленту" мобильного экрана.
const mobileRecentLimit = 5

// GetMobileHome — агрегированный главный экран мобильного приложения.
//
// GET /api/v1/mobile/home
//
// Собирает за один запрос:
//   - счётчик непрочитанных уведомлений + последние N,
//   - назначенные на пользователя задачи (top N),
//   - ближайшие дедлайны по доске пользователя.
//
// Отказоустойчивость: главный экран не должен падать целиком из-за одного
// недоступного сервиса — каждый под-вызов деградирует в пустую секцию,
// а не роняет весь ответ.
func (h *Handler) GetMobileHome(c *gin.Context) {
	userID := ginInt64(c, "userId")
	role := ginString(c, "role")
	departmentID := ginInt64(c, "departmentId")

	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := outgoingCtx(c)

	home := gin.H{
		"user_id":       userID,
		"role":          role,
		"department_id": departmentID,
	}

	// --- Уведомления: непрочитанные (счётчик + последние) ---
	if nResp, err := h.notificationClient.ListNotifications(ctx, &notificationv1.ListNotificationsRequest{
		UserId:     userID,
		OnlyUnread: true,
		Page:       1,
		PageSize:   mobileRecentLimit,
	}); err == nil && nResp != nil {
		home["notifications"] = gin.H{
			"unread_count": nResp.TotalCount,
			"recent":       nResp.Notifications,
		}
	} else {
		home["notifications"] = gin.H{"unread_count": 0, "recent": nil}
	}

	// --- Мои задачи (назначенные на пользователя) ---
	if tResp, err := h.taskClient.GetMyTasks(ctx, &taskv1.GetMyTasksRequest{
		OnlyAssigned: true,
		Page:         1,
		PageSize:     mobileRecentLimit,
	}); err == nil && tResp != nil {
		home["tasks"] = gin.H{
			"items": tResp.Tasks,
			"total": tResp.TotalCount,
		}
	} else {
		home["tasks"] = gin.H{"items": nil, "total": 0}
	}

	// --- Ближайшие дедлайны (по доске пользователя) ---
	// Доска резолвится по роли (student: team->project->board, иначе — мои доски).
	// Если доску определить не удалось — отдаём пустой список, экран не ломаем.
	if boardID, err := h.resolveBoardID(c); err == nil && boardID > 0 {
		if dResp, derr := h.taskClient.GetUpcomingDeadlines(ctx, &taskv1.GetUpcomingDeadlinesRequest{
			BoardId:   boardID,
			UserId:    userID,
			DaysAhead: 7,
			Page:      1,
			PageSize:  mobileRecentLimit,
		}); derr == nil && dResp != nil {
			home["upcoming_deadlines"] = gin.H{
				"items": dResp.Tasks,
				"total": dResp.TotalCount,
			}
		} else {
			home["upcoming_deadlines"] = gin.H{"items": nil, "total": 0}
		}
	} else {
		home["upcoming_deadlines"] = gin.H{"items": nil, "total": 0}
	}

	c.JSON(http.StatusOK, home)
}

// RegisterDevice — регистрация push-токена устройства (FCM).
//
// POST /api/v1/mobile/devices  { "token": "...", "platform": "android" }
func (h *Handler) RegisterDevice(c *gin.Context) {
	if ginInt64(c, "userId") == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var body struct {
		Token    string `json:"token" binding:"required"`
		Platform string `json:"platform"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	resp, err := h.notificationClient.RegisterDevice(outgoingCtx(c), &notificationv1.RegisterDeviceRequest{
		Token:    body.Token,
		Platform: body.Platform,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// UnregisterDevice — удаление push-токена устройства (например, при logout).
//
// DELETE /api/v1/mobile/devices  { "token": "..." }
func (h *Handler) UnregisterDevice(c *gin.Context) {
	if ginInt64(c, "userId") == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var body struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	if _, err := h.notificationClient.UnregisterDevice(outgoingCtx(c), &notificationv1.UnregisterDeviceRequest{
		Token: body.Token,
	}); err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ListDevices — список зарегистрированных устройств текущего пользователя.
//
// GET /api/v1/mobile/devices
func (h *Handler) ListDevices(c *gin.Context) {
	if ginInt64(c, "userId") == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	resp, err := h.notificationClient.ListDevices(outgoingCtx(c), &notificationv1.ListDevicesRequest{})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
