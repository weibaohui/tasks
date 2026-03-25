package feishu

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

// messageHandler handles incoming Feishu messages
type messageHandler struct {
	channel *Channel
}

// newMessageHandler creates a new message handler
func newMessageHandler(channel *Channel) *messageHandler {
	return &messageHandler{channel: channel}
}

// onMessageReceive handles received messages
func (h *messageHandler) onMessageReceive(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	c := h.channel
	c.logger.Info("Feishu 收到事件回调")

	if event == nil || event.Event == nil {
		c.logger.Debug("Feishu event is empty")
		return nil
	}

	ev := event.Event
	message := ev.Message
	sender := ev.Sender

	if message == nil || sender == nil {
		c.logger.Warn("Feishu message or sender is empty")
		return nil
	}

	// Message deduplication check
	messageID := *message.MessageId
	if !c.processedMsgIDs.add(messageID) {
		c.logger.Debug("Duplicate Feishu message, ignoring", zap.String("message_id", messageID))
		return nil
	}

	// Skip bot messages
	if sender.SenderType != nil && *sender.SenderType == "bot" {
		c.logger.Debug("Skipping bot message")
		return nil
	}

	// Get sender ID
	senderID := "unknown"
	if sender.SenderId != nil && sender.SenderId.OpenId != nil {
		senderID = *sender.SenderId.OpenId
	}

	// Get chat ID
	chatID := ""
	if message.ChatId != nil {
		chatID = *message.ChatId
	}

	// Get chat type
	chatType := "p2p"
	if message.ChatType != nil {
		chatType = *message.ChatType
	}

	// Get message type
	msgType := ""
	if message.MessageType != nil {
		msgType = *message.MessageType
	}

	// Parse message content
	content := h.parseMessageContent(message)
	if content == "" {
		return nil
	}

	// Determine reply target
	replyTo := chatID
	if chatType == "p2p" {
		replyTo = senderID
	}

	// Check user whitelist
	if len(c.config.AllowFrom) > 0 {
		allowed := false
		for _, u := range c.config.AllowFrom {
			if senderID == u {
				allowed = true
				break
			}
		}
		if !allowed {
			c.logger.Warn("Feishu message sender not in whitelist", zap.String("sender", senderID), zap.Strings("whitelist", c.config.AllowFrom))
			return nil
		}
	}

	c.logger.Info("Received Feishu message",
		zap.String("sender", senderID),
		zap.String("chat_id", chatID),
		zap.String("reply_to", replyTo),
		zap.String("content", content),
	)

	// Add reaction emoji to indicate message is being processed
	go c.addReactionAndSave(messageID, "OnIt")

	// Publish message to bus
	// Record app_id in Metadata for message routing
	// Record channel_id for loading channel-bound agent config
	c.bus.PublishInbound(&bus.InboundMessage{
		Channel:   "feishu",
		ChatID:    replyTo,
		SenderID:  senderID,
		Content:   content,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"message_id":   messageID,
			"chat_type":    chatType,
			"msg_type":     msgType,
			"chat_id":      chatID,
			"sender_id":    senderID,
			"app_id":       c.config.AppID,
			"channel_id":   c.config.ChannelID,
			"channel_code": c.config.ChannelCode,
			"agent_code":   c.config.AgentCode,
			"user_code":    c.config.UserCode,
		},
	})

	return nil
}

// onReactionCreated handles message reaction creation events
func (h *messageHandler) onReactionCreated(ctx context.Context, event *larkim.P2MessageReactionCreatedV1) error {
	c := h.channel
	if event != nil && event.Event != nil {
		ev := event.Event
		emojiType := ""
		if ev.ReactionType != nil && ev.ReactionType.EmojiType != nil {
			emojiType = *ev.ReactionType.EmojiType
		}
		c.logger.Debug("Received Feishu reaction creation event",
			zap.String("message_id", *ev.MessageId),
			zap.String("emoji", emojiType),
		)
	}
	return nil
}

// onReactionDeleted handles message reaction deletion events
func (h *messageHandler) onReactionDeleted(ctx context.Context, event *larkim.P2MessageReactionDeletedV1) error {
	c := h.channel
	if event != nil && event.Event != nil {
		ev := event.Event
		emojiType := ""
		if ev.ReactionType != nil && ev.ReactionType.EmojiType != nil {
			emojiType = *ev.ReactionType.EmojiType
		}
		c.logger.Debug("Received Feishu reaction deletion event",
			zap.String("message_id", *ev.MessageId),
			zap.String("emoji", emojiType),
		)
	}
	return nil
}

// parseMessageContent parses message content based on message type
func (h *messageHandler) parseMessageContent(message *larkim.EventMessage) string {
	if message == nil || message.MessageType == nil {
		return ""
	}

	msgType := *message.MessageType

	switch msgType {
	case "text":
		if message.Content == nil {
			return ""
		}
		// Parse JSON formatted text content
		var contentMap map[string]interface{}
		if err := json.Unmarshal([]byte(*message.Content), &contentMap); err == nil {
			if text, ok := contentMap["text"].(string); ok {
				return strings.TrimSpace(text)
			}
		}
		return strings.TrimSpace(*message.Content)
	case "image":
		return "[Image]"
	case "audio":
		return "[Voice]"
	case "file":
		return "[File]"
	case "sticker":
		return "[Sticker]"
	default:
		return "[" + msgType + "]"
	}
}
