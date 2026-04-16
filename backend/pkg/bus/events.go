package bus

import "time"

// InboundMessage represents a message received from a chat channel
type InboundMessage struct {
	Channel   string         `json:"channel"`   // feishu, dingtalk, wechat, etc.
	SenderID  string         `json:"sender_id"` // User identifier
	ChatID    string         `json:"chat_id"`   // Chat/channel identifier
	Content   string         `json:"content"`   // Message text
	Timestamp time.Time      `json:"timestamp"` // Timestamp
	Media     []string       `json:"media"`     // Media URLs
	Metadata  map[string]any `json:"metadata"`  // Channel-specific data
}

// SessionKey returns a unique identifier for the session
func (m *InboundMessage) SessionKey() string {
	// 需求派发的消息按 requirement_id 隔离 session，使不同需求可以并发处理
	if m.Metadata != nil {
		if reqID, ok := m.Metadata["requirement_id"].(string); ok && reqID != "" {
			return m.Channel + ":" + m.ChatID + ":req:" + reqID
		}
	}
	return m.Channel + ":" + m.ChatID
}

// OutboundMessage represents a message to be sent to a chat channel
type OutboundMessage struct {
	Channel  string         `json:"channel"`
	ChatID   string         `json:"chat_id"`
	Content  string         `json:"content"`
	ReplyTo  string         `json:"reply_to,omitempty"`
	Media    []string       `json:"media,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// StreamChunk represents a chunk of streaming output
type StreamChunk struct {
	Channel string `json:"channel"`
	ChatID  string `json:"chat_id"`
	Delta   string `json:"delta"`   // Incremental content
	Content string `json:"content"` // Accumulated content
	Done    bool   `json:"done"`    // Whether streaming is complete
}

// NewInboundMessage creates a new inbound message
func NewInboundMessage(channel, senderID, chatID, content string) *InboundMessage {
	return &InboundMessage{
		Channel:   channel,
		SenderID:  senderID,
		ChatID:    chatID,
		Content:   content,
		Timestamp: time.Now(),
		Media:     []string{},
		Metadata:  make(map[string]any),
	}
}

// NewOutboundMessage creates a new outbound message
func NewOutboundMessage(channel, chatID, content string) *OutboundMessage {
	return &OutboundMessage{
		Channel:  channel,
		ChatID:   chatID,
		Content:  content,
		Media:    []string{},
		Metadata: make(map[string]any),
	}
}

// NewStreamChunk creates a new stream chunk
func NewStreamChunk(channel, chatID, delta, content string, done bool) *StreamChunk {
	return &StreamChunk{
		Channel: channel,
		ChatID:  chatID,
		Delta:   delta,
		Content: content,
		Done:    done,
	}
}
