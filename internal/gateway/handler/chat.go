package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	chatv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/chat/v1"
	"google.golang.org/grpc/metadata"
)

// wsOutgoingCtx строит контекст с metadata пользователя для вызовов chat_service
// из WebSocket read-цикла. В отличие от outgoingCtx использует context.Background(),
// т.к. соединение живёт дольше исходного HTTP-запроса.
func wsOutgoingCtx(c *gin.Context) context.Context {
	userID := ginInt64(c, "userId")
	role := ginString(c, "role")
	universityID := ginInt64(c, "universityId")
	departmentID := ginInt64(c, "departmentId")
	return metadata.AppendToOutgoingContext(
		context.Background(),
		"x-user-id", strconv.FormatInt(userID, 10),
		"x-user-role", role,
		"x-university-id", strconv.FormatInt(universityID, 10),
		"x-department-id", strconv.FormatInt(departmentID, 10),
	)
}

// wsUpgrader апгрейдит HTTP в WebSocket. Origin-проверку оставляем шлюзу/nginx
// (CORS уже применяется на уровне роутера), поэтому здесь разрешаем все.
var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// outgoingWSMessage — формат события, отправляемого клиентам по WebSocket.
type outgoingWSMessage struct {
	Type    string          `json:"type"`
	Message *chatv1.Message `json:"message"`
}

// CreateConversation — POST /api/v1/chat/conversations
func (h *Handler) CreateConversation(c *gin.Context) {
	if ginInt64(c, "userId") == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var body struct {
		Type           string  `json:"type"`
		Title          string  `json:"title"`
		ParticipantIDs []int64 `json:"participant_ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	resp, err := h.chatClient.CreateConversation(outgoingCtx(c), &chatv1.CreateConversationRequest{
		Type:           body.Type,
		Title:          body.Title,
		ParticipantIds: body.ParticipantIDs,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ListConversations — GET /api/v1/chat/conversations
func (h *Handler) ListConversations(c *gin.Context) {
	resp, err := h.chatClient.ListConversations(outgoingCtx(c), &chatv1.ListConversationsRequest{})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetConversation — GET /api/v1/chat/conversations/:id
func (h *Handler) GetConversation(c *gin.Context) {
	convID := parseInt64(c.Param("id"))
	resp, err := h.chatClient.GetConversation(outgoingCtx(c), &chatv1.GetConversationRequest{
		ConversationId: convID,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ListMessages — GET /api/v1/chat/conversations/:id/messages
func (h *Handler) ListMessages(c *gin.Context) {
	convID := parseInt64(c.Param("id"))
	page := int32(1)
	pageSize := int32(30)
	if p := c.Query("page"); p != "" {
		if v, _ := strconv.ParseInt(p, 10, 32); v > 0 {
			page = int32(v)
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if v, _ := strconv.ParseInt(ps, 10, 32); v > 0 {
			pageSize = int32(v)
		}
	}
	resp, err := h.chatClient.ListMessages(outgoingCtx(c), &chatv1.ListMessagesRequest{
		ConversationId: convID,
		Page:           page,
		PageSize:       pageSize,
	})
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// SendMessage — POST /api/v1/chat/conversations/:id/messages
func (h *Handler) SendMessage(c *gin.Context) {
	convID := parseInt64(c.Param("id"))
	var body struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return
	}
	msg, err := h.deliverMessage(outgoingCtx(c), convID, body.Content)
	if err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": msg})
}

// MarkChatRead — POST /api/v1/chat/conversations/:id/read
func (h *Handler) MarkChatRead(c *gin.Context) {
	convID := parseInt64(c.Param("id"))
	var body struct {
		LastReadMessageID int64 `json:"last_read_message_id"`
	}
	_ = c.ShouldBindJSON(&body)
	if _, err := h.chatClient.MarkRead(outgoingCtx(c), &chatv1.MarkReadRequest{
		ConversationId:    convID,
		LastReadMessageId: body.LastReadMessageID,
	}); err != nil {
		MapGRPCError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// deliverMessage персистит сообщение через chat_service и рассылает его в realtime
// всем участникам беседы по WebSocket. Используется и REST-, и WS-путём.
func (h *Handler) deliverMessage(ctx context.Context, convID int64, content string) (*chatv1.Message, error) {
	resp, err := h.chatClient.SendMessage(ctx, &chatv1.SendMessageRequest{
		ConversationId: convID,
		Content:        content,
	})
	if err != nil {
		return nil, err
	}
	if payload, mErr := json.Marshal(outgoingWSMessage{Type: "message", Message: resp.Message}); mErr == nil {
		h.chatHub.SendToUsers(resp.ParticipantIds, payload)
	}
	return resp.Message, nil
}

// ChatWebSocket — GET /api/v1/chat/ws
//
// Realtime-канал чата. Аутентификация — через AuthMiddleware (заголовок
// Authorization: Bearer <token>; Flutter поддерживает заголовки при WS-подключении).
// Входящие кадры: {"conversation_id": <id>, "content": "<text>"}.
func (h *Handler) ChatWebSocket(c *gin.Context) {
	userID := ginInt64(c, "userId")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return // upgrade сам пишет ответ
	}

	client := h.chatHub.NewClient(conn, userID)
	h.chatHub.Register(client)
	go client.WritePump()
	defer func() {
		h.chatHub.Unregister(client)
		_ = conn.Close()
	}()

	client.SetupRead()

	// Контекст с metadata пользователя для вызовов chat_service из read-цикла.
	ctx := wsOutgoingCtx(c)

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var in struct {
			ConversationID int64  `json:"conversation_id"`
			Content        string `json:"content"`
		}
		if json.Unmarshal(raw, &in) != nil || in.ConversationID == 0 || in.Content == "" {
			continue
		}
		// Ошибки доставки (например, не участник) не рвут соединение — просто пропускаем кадр.
		_, _ = h.deliverMessage(ctx, in.ConversationID, in.Content)
	}
}
