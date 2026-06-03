package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	chatv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/chat/v1"
	taskv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/task/v1"
	"github.com/lucky720s/diplomaflow/pkg/realtime"
)

// DispatchRealtime фанит событие из Redis всем WS-клиентам, подписанным на его
// топик. Вызывается из фонового подписчика gateway (см. main.go).
func (h *Handler) DispatchRealtime(ev realtime.Event) {
	if ev.Topic == "" {
		return
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.chatHub.SendToTopic(ev.Topic, data)
}

// RealtimeWebSocket — GET /api/v1/realtime/ws
//
// Единый realtime-канал (доска, уведомления). Аутентификация — AuthMiddleware
// (Authorization: Bearer <token>). При подключении клиент автоматически
// подписан на свой топик user:{id} (туда идут уведомления). На доски/беседы
// клиент подписывается явно, доступ проверяется через соответствующий сервис.
//
// Кадры клиент→сервер:
//
//	{ "action": "subscribe",   "topic": "board:42" }
//	{ "action": "unsubscribe", "topic": "board:42" }
func (h *Handler) RealtimeWebSocket(c *gin.Context) {
	userID := ginInt64(c, "userId")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := h.chatHub.NewClient(conn, userID)
	h.chatHub.Register(client)
	// Личный топик — для адресных событий (уведомления).
	h.chatHub.Join(client, realtime.UserTopic(userID))
	go client.WritePump()
	defer func() {
		h.chatHub.Unregister(client)
		_ = conn.Close()
	}()

	client.SetupRead()
	ctx := wsOutgoingCtx(c)

	for {
		_, raw, readErr := conn.ReadMessage()
		if readErr != nil {
			return
		}
		var msg struct {
			Action string `json:"action"`
			Topic  string `json:"topic"`
		}
		if json.Unmarshal(raw, &msg) != nil || msg.Topic == "" {
			continue
		}

		switch msg.Action {
		case "subscribe":
			if h.canSubscribe(ctx, userID, msg.Topic) {
				h.chatHub.Join(client, msg.Topic)
			}
		case "unsubscribe":
			h.chatHub.Leave(client, msg.Topic)
		}
	}
}

// canSubscribe проверяет, имеет ли пользователь право на топик.
// Доступ к доскам/беседам валидируется через соответствующий gRPC-сервис.
func (h *Handler) canSubscribe(ctx context.Context, userID int64, topic string) bool {
	kind, id, ok := parseTopic(topic)
	if !ok {
		return false
	}
	switch kind {
	case "user":
		// Только собственный топик.
		return id == userID
	case "board":
		_, err := h.taskClient.GetBoard(ctx, &taskv1.GetBoardRequest{BoardId: id})
		return err == nil
	case "conversation":
		_, err := h.chatClient.GetConversation(ctx, &chatv1.GetConversationRequest{ConversationId: id})
		return err == nil
	default:
		return false
	}
}

// parseTopic разбирает "kind:id".
func parseTopic(topic string) (kind string, id int64, ok bool) {
	parts := strings.SplitN(topic, ":", 2)
	if len(parts) != 2 {
		return "", 0, false
	}
	v, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || v <= 0 {
		return "", 0, false
	}
	return parts[0], v, true
}
