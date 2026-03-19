/**
 * WebSocket Handler
 * 处理 WebSocket 连接和消息
 */
package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/bus"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocketHandler WebSocket处理器
type WebSocketHandler struct {
	eventBus *bus.EventBus
	clients  map[string]map[*Client]bool
	mu       sync.RWMutex
}

// Client WebSocket客户端
type Client struct {
	conn    *websocket.Conn
	traceID string
	send    chan []byte
}

// NewWebSocketHandler 创建WebSocket处理器
func NewWebSocketHandler(eventBus *bus.EventBus) *WebSocketHandler {
	return &WebSocketHandler{
		eventBus: eventBus,
		clients:  make(map[string]map[*Client]bool),
	}
}

// WSMessage WebSocket消息
type WSMessage struct {
	Type      string      `json:"type"`
	TraceID   string      `json:"trace_id"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// HandleWebSocket 处理WebSocket连接
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	traceID := r.URL.Query().Get("trace_id")
	if traceID == "" {
		http.Error(w, "trace_id is required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		conn:    conn,
		traceID: traceID,
		send:    make(chan []byte, 256),
	}

	h.registerClient(client)

	defer func() {
		h.unregisterClient(client)
		conn.Close()
	}()

	go h.writePump(client)
	h.readPump(client)
}

func (h *WebSocketHandler) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[client.traceID] == nil {
		h.clients[client.traceID] = make(map[*Client]bool)
	}
	h.clients[client.traceID][client] = true

	log.Printf("Client registered for trace_id: %s", client.traceID)
}

func (h *WebSocketHandler) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[client.traceID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.clients, client.traceID)
		}
	}

	close(client.send)
	log.Printf("Client unregistered for trace_id: %s", client.traceID)
}

func (h *WebSocketHandler) writePump(client *Client) {
	for message := range client.send {
		if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("Write error: %v", err)
			return
		}
	}
}

func (h *WebSocketHandler) readPump(client *Client) {
	defer func() {
		h.unregisterClient(client)
		client.conn.Close()
	}()

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Read error: %v", err)
			}
			break
		}

		log.Printf("Received message from client: %s", message)

		// 处理客户端消息（如果需要）
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err == nil {
			log.Printf("Parsed message: %+v", msg)
		}
	}
}

// broadcastToTrace 向指定 trace_id 的所有客户端广播消息
func (h *WebSocketHandler) broadcastToTrace(traceID string, message *WSMessage) {
	h.mu.RLock()
	clients, ok := h.clients[traceID]
	h.mu.RUnlock()

	if !ok {
		return
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	h.mu.RLock()
	for client := range clients {
		select {
		case client.send <- data:
		default:
			go h.unregisterClient(client)
		}
	}
	h.mu.RUnlock()
}

// SubscribeToEvents 订阅领域事件
func (h *WebSocketHandler) SubscribeToEvents() {
	h.eventBus.Subscribe("TaskCreated", func(event domain.DomainEvent) {
		h.broadcastToTrace(event.TraceID().String(), &WSMessage{
			Type:      "TaskCreated",
			TraceID:   event.TraceID().String(),
			Data:      nil,
			Timestamp: event.Timestamp(),
		})
	})

	h.eventBus.Subscribe("TaskStarted", func(event domain.DomainEvent) {
		h.broadcastToTrace(event.TraceID().String(), &WSMessage{
			Type:      "TaskStarted",
			TraceID:   event.TraceID().String(),
			Data:      nil,
			Timestamp: event.Timestamp(),
		})
	})

	h.eventBus.Subscribe("TaskCompleted", func(event domain.DomainEvent) {
		h.broadcastToTrace(event.TraceID().String(), &WSMessage{
			Type:      "TaskCompleted",
			TraceID:   event.TraceID().String(),
			Data:      nil,
			Timestamp: event.Timestamp(),
		})
	})

	h.eventBus.Subscribe("TaskFailed", func(event domain.DomainEvent) {
		h.broadcastToTrace(event.TraceID().String(), &WSMessage{
			Type:      "TaskFailed",
			TraceID:   event.TraceID().String(),
			Data:      nil,
			Timestamp: event.Timestamp(),
		})
	})

	h.eventBus.Subscribe("TaskCancelled", func(event domain.DomainEvent) {
		h.broadcastToTrace(event.TraceID().String(), &WSMessage{
			Type:      "TaskCancelled",
			TraceID:   event.TraceID().String(),
			Data:      nil,
			Timestamp: event.Timestamp(),
		})
	})

	h.eventBus.Subscribe("TaskProgressUpdated", func(event domain.DomainEvent) {
		if progressEvent, ok := event.(*domain.TaskProgressUpdatedEvent); ok {
			h.broadcastToTrace(event.TraceID().String(), &WSMessage{
				Type:      "TaskProgressUpdated",
				TraceID:   event.TraceID().String(),
				Data:      progressEvent.GetProgress().ToMap(),
				Timestamp: event.Timestamp(),
			})
		}
	})
}
