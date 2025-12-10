package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type NotificationMessage struct {
	Type       string `json:"type"`
	UserID     int64  `json:"user_id"`
	MessageID  int64  `json:"message_id"`
	FromEmail  string `json:"from_email,omitempty"`
	FromID     int64  `json:"from_id,omitempty"`
	Subject    string `json:"subject,omitempty"`
	Preview    string `json:"preview,omitempty"`
	Timestamp  int64  `json:"timestamp"`
	IsInternal bool   `json:"is_internal"`
}

type Client struct {
	UserID string
	Conn   *websocket.Conn
}

type WebSocketManager struct {
	clients  map[string][]*websocket.Conn
	mu       sync.RWMutex
	redisSub *redis.PubSub
	redis    *redis.Client
}

func NewWebSocketManager(redisClient *redis.Client) *WebSocketManager {
	manager := &WebSocketManager{
		clients: make(map[string][]*websocket.Conn),
		redis:   redisClient,
	}

	if redisClient != nil {
		manager.redisSub = redisClient.Subscribe(context.Background(), "notifications")
		go manager.listenRedis()
	}

	return manager
}

func (m *WebSocketManager) listenRedis() {
	if m.redisSub == nil {
		return
	}

	ch := m.redisSub.Channel()
	for msg := range ch {
		var notification NotificationMessage
		if err := json.Unmarshal([]byte(msg.Payload), &notification); err != nil {
			slog.Error("Failed to parse notification", "error", err)
			continue
		}

		userID := strconv.FormatInt(notification.UserID, 10)
		m.SendToUser(userID, []byte(msg.Payload))
	}
}

func (m *WebSocketManager) RegisterClient(userID string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.clients[userID]; !exists {
		m.clients[userID] = []*websocket.Conn{}
	}

	m.clients[userID] = append(m.clients[userID], conn)
	slog.Debug("WebSocket client registered", "user_id", userID, "connections", len(m.clients[userID]))
}

func (m *WebSocketManager) UnregisterClient(userID string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if connections, ok := m.clients[userID]; ok {
		for i, c := range connections {
			if c == conn {
				m.clients[userID] = append(connections[:i], connections[i+1:]...)
				break
			}
		}

		if len(m.clients[userID]) == 0 {
			delete(m.clients, userID)
		}
	}
}

func (m *WebSocketManager) SendToUser(userID string, message []byte) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if connections, ok := m.clients[userID]; ok {
		slog.Debug("Sending notification to user", "user_id", userID, "connections", len(connections))

		var failedConnections []*websocket.Conn

		for _, conn := range connections {
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				slog.Error("Failed to send WebSocket message",
					"user_id", userID,
					"error", err,
				)
				failedConnections = append(failedConnections, conn)
			}
		}

		if len(failedConnections) > 0 {
			go func() {
				m.mu.Lock()
				defer m.mu.Unlock()

				for _, failedConn := range failedConnections {
					for i, conn := range m.clients[userID] {
						if conn == failedConn {
							m.clients[userID] = append(m.clients[userID][:i], m.clients[userID][i+1:]...)
							failedConn.Close()
							break
						}
					}
				}

				if len(m.clients[userID]) == 0 {
					delete(m.clients, userID)
				}
			}()
		}

		return nil
	}

	slog.Debug("No active WebSocket connections for user", "user_id", userID)
	return fmt.Errorf("no active connections for user %s", userID)
}

func (m *WebSocketManager) Shutdown() {
	if m.redisSub != nil {
		m.redisSub.Close()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for userID, connections := range m.clients {
		for _, conn := range connections {
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, "Server shutdown"))
			conn.Close()
		}
		delete(m.clients, userID)
	}
}

func (m *WebSocketManager) GetConnectedUsers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := make([]string, 0, len(m.clients))
	for userID := range m.clients {
		users = append(users, userID)
	}
	return users
}

func (m *WebSocketManager) GetConnectionCount(userID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if connections, ok := m.clients[userID]; ok {
		return len(connections)
	}
	return 0
}
