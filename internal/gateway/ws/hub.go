// Package ws — in-memory WebSocket-хаб для realtime-доставки чата на gateway.
//
// Назначение: держать активные WS-соединения пользователей и рассылать им
// сообщения. Источник правды (история) — chat_service; хаб только транспорт.
//
// Ограничение: хаб in-memory и работает в пределах одного инстанса gateway.
// Для горизонтального масштабирования сюда добавляется Redis pub/sub, но для
// текущего объёма (один gateway) этого достаточно.
package ws

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	sendBuffer = 32
)

// Hub хранит соединения, сгруппированные по userID (один пользователь может быть
// подключён с нескольких устройств).
type Hub struct {
	mu      sync.RWMutex
	clients map[int64]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[int64]map[*Client]struct{})}
}

// Client — одно WS-соединение пользователя с собственной очередью на запись.
type Client struct {
	UserID int64
	conn   *websocket.Conn
	send   chan []byte
	hub    *Hub
}

func (h *Hub) NewClient(conn *websocket.Conn, userID int64) *Client {
	return &Client{
		UserID: userID,
		conn:   conn,
		send:   make(chan []byte, sendBuffer),
		hub:    h,
	}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.UserID] == nil {
		h.clients[c.UserID] = make(map[*Client]struct{})
	}
	h.clients[c.UserID][c] = struct{}{}
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.clients[c.UserID]; ok {
		if _, ok := set[c]; ok {
			delete(set, c)
			close(c.send)
		}
		if len(set) == 0 {
			delete(h.clients, c.UserID)
		}
	}
}

// SendToUsers кладёт payload в очередь всем подключённым клиентам из списка.
// Если очередь клиента переполнена (медленный потребитель) — сообщение для него
// дропается, чтобы не блокировать рассылку.
func (h *Hub) SendToUsers(userIDs []int64, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, uid := range userIDs {
		for c := range h.clients[uid] {
			select {
			case c.send <- payload:
			default:
			}
		}
	}
}

// Conn возвращает нижележащее соединение для read-цикла на стороне хендлера.
func (c *Client) Conn() *websocket.Conn { return c.conn }

// WritePump перекачивает очередь send в сокет и шлёт ping для поддержания
// соединения. Запускается в отдельной горутине на каждого клиента.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SetupRead настраивает таймауты чтения и pong-handler перед read-циклом.
func (c *Client) SetupRead() {
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
}
