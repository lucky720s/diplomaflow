// Package realtime — общий контракт realtime-событий поверх Redis Pub/Sub.
//
// Модель: сервисы публикуют доменные события в один Redis-канал (Channel),
// gateway подписан на него и фанит события в WebSocket-клиентов по топику.
// Realtime — best-effort слой поверх REST: источник правды остаётся в БД,
// при реконнекте клиент дотягивает состояние обычными запросами.
package realtime

import (
	"encoding/json"
	"fmt"
	"time"
)

// Channel — единый Redis Pub/Sub-канал для всех realtime-событий.
const Channel = "realtime:events"

// Типы событий (envelope.Type).
const (
	EventTaskCreated         = "task.created"
	EventTaskUpdated         = "task.updated"
	EventTaskMoved           = "task.moved"
	EventTaskDeleted         = "task.deleted"
	EventCommentCreated      = "comment.created"
	EventBoardUpdated        = "board.updated"
	EventNotificationCreated = "notification.created"
	EventChatMessage         = "chat.message"
)

// Event — конверт realtime-события. Доставляется клиенту как есть (JSON).
type Event struct {
	Type    string          `json:"type"`
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Ts      int64           `json:"ts"`
}

// NewEvent собирает событие, маршаля payload в JSON.
func NewEvent(eventType, topic string, payload any) (Event, error) {
	var raw json.RawMessage
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return Event{}, err
		}
		raw = b
	}
	return Event{
		Type:    eventType,
		Topic:   topic,
		Payload: raw,
		Ts:      time.Now().UTC().UnixMilli(),
	}, nil
}

// ---- Топики ----

func UserTopic(id int64) string         { return fmt.Sprintf("user:%d", id) }
func BoardTopic(id int64) string        { return fmt.Sprintf("board:%d", id) }
func ProjectTopic(id int64) string      { return fmt.Sprintf("project:%d", id) }
func ConversationTopic(id int64) string { return fmt.Sprintf("conversation:%d", id) }
