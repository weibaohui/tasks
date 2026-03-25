package feishu

import (
	"context"
	"sync"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/larksuite/oapi-sdk-go/v3/ws"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

// Config holds the Feishu channel configuration
type Config struct {
	AppID             string   `json:"app_id"`
	AppSecret         string   `json:"app_secret"`
	EncryptKey        string   `json:"encrypt_key"`
	VerificationToken string   `json:"verification_token"`
	AllowFrom         []string `json:"allow_from"`
	ChannelCode       string   `json:"channel_code"` // Channel code from database
	ChannelID         string   `json:"channel_id"`   // Channel ID from database
	AgentCode         string   `json:"agent_code"`   // Bound agent code
	UserCode          string   `json:"user_code"`    // Bound user code (from agent)
}

// Channel implements the Channel interface for Feishu
// Uses WebSocket long connection to receive messages, HTTP API to send messages
type Channel struct {
	bus     *bus.MessageBus
	name    string
	config  *Config
	logger  *zap.Logger
	running bool

	// Feishu client
	client *lark.Client

	// WebSocket client
	wsClient *ws.Client

	// Background task management
	bgTasks sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc

	// Message deduplication cache
	processedMsgIDs *syncMap

	// Reaction cache: message_id -> reactionInfo
	reactionCache map[string]*reactionInfo
	reactionMu    sync.RWMutex

	// Event handler
	eventHandler *dispatcher.EventDispatcher
}

// reactionInfo holds message reaction information
type reactionInfo struct {
	messageID  string
	reactionID string
}

// syncMap is a thread-safe map with size limit for deduplication
type syncMap struct {
	data    map[string]time.Time
	mu      sync.RWMutex
	maxSize int
}

// newSyncMap creates a new sync map
func newSyncMap(maxSize int) *syncMap {
	return &syncMap{
		data:    make(map[string]time.Time),
		maxSize: maxSize,
	}
}

// add adds an element and returns false if it already exists
func (m *syncMap) add(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.data[key]; exists {
		return false
	}

	m.data[key] = time.Now()

	// Clean up expired data when size exceeds limit
	if len(m.data) > m.maxSize {
		toDelete := int(float64(m.maxSize) * 0.2)
		for k := range m.data {
			if toDelete <= 0 {
				break
			}
			delete(m.data, k)
			toDelete--
		}
	}

	return true
}

// MessageEvent wraps Feishu message event
type MessageEvent struct {
	Message   *larkim.EventMessage
	Sender    *larkim.EventSender
	ChatID    string
	ChatType  string
	MsgType   string
	Content   string
	MessageID string
	SenderID  string
}
