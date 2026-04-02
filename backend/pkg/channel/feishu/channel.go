package feishu

import (
	"context"
	"fmt"
	"strings"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	"github.com/larksuite/oapi-sdk-go/v3/ws"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

// ChannelTypeFeishu is the type identifier for Feishu channel
const ChannelTypeFeishu = "feishu"

// NewChannel creates a new Feishu channel
func NewChannel(config *Config, messageBus *bus.MessageBus, logger *zap.Logger) *Channel {
	if logger == nil {
		logger = zap.NewNop()
	}
	// Use app_id as part of channel name to ensure uniqueness
	channelName := "feishu"
	if config.AppID != "" {
		channelName = fmt.Sprintf("feishu_%s", config.AppID)
	}
	return &Channel{
		bus:             messageBus,
		name:            channelName,
		config:          config,
		logger:          logger,
		processedMsgIDs: newSyncMap(1000),
		reactionCache:   make(map[string]*reactionInfo),
	}
}

// Name returns the channel name
func (c *Channel) Name() string {
	return c.name
}

// Type returns the channel type
func (c *Channel) Type() string {
	return ChannelTypeFeishu
}

// Bus returns the message bus
func (c *Channel) Bus() *bus.MessageBus {
	return c.bus
}

// Start starts the Feishu channel
func (c *Channel) Start(ctx context.Context) error {
	if c.config.AppID == "" || c.config.AppSecret == "" {
		c.logger.Error("Feishu app_id and app_secret not configured")
		return fmt.Errorf("incomplete Feishu configuration")
	}

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.running = true

	// Create Feishu client (for sending messages)
	c.client = lark.NewClient(c.config.AppID, c.config.AppSecret)

	// Create event handler
	handler := newMessageHandler(c)
	c.eventHandler = dispatcher.NewEventDispatcher(
		c.config.VerificationToken,
		c.config.EncryptKey,
	).OnP2MessageReceiveV1(handler.onMessageReceive).
		OnP2MessageReactionCreatedV1(handler.onReactionCreated).
		OnP2MessageReactionDeletedV1(handler.onReactionDeleted)

	// Create WebSocket client
	c.wsClient = ws.NewClient(c.config.AppID, c.config.AppSecret,
		ws.WithEventHandler(c.eventHandler),
		ws.WithLogLevel(larkcore.LogLevelInfo),
	)

	// Subscribe to outbound messages
	c.bus.SubscribeOutbound("feishu", func(msg *bus.OutboundMessage) error {
		// Check if message belongs to this channel (via app_id matching)
		if msg.Metadata != nil {
			if targetAppID, ok := msg.Metadata["app_id"].(string); ok && targetAppID != "" {
				if targetAppID != c.config.AppID {
					// Message belongs to another Feishu channel, skip
					return nil
				}
			}
		}
		if err := c.Send(msg); err != nil {
			// Cross-app open_id error is common when bot and user are in different apps - skip silently
			if !isCrossAppError(err) {
				c.logger.Error("Failed to send Feishu message", zap.Error(err))
				return err
			}
			// Silently skip cross-app errors
		}

		// Delete reaction after message is sent successfully
		if msg.Metadata != nil {
			if replyToMsgID, ok := msg.Metadata["reply_to_message_id"].(string); ok && replyToMsgID != "" {
				c.deleteReactionFromCache(replyToMsgID)
			}
		}

		return nil
	})

	c.logger.Info("Feishu channel started",
		zap.String("app_id", c.config.AppID),
	)

	// Start WebSocket client (with reconnection)
	c.bgTasks.Add(1)
	go c.runWebSocketClient()

	return nil
}

// runWebSocketClient runs the WebSocket client with reconnection
func (c *Channel) runWebSocketClient() {
	defer c.bgTasks.Done()

	for c.running {
		err := c.wsClient.Start(c.ctx)
		if err != nil {
			if c.ctx.Err() != nil {
				// Context cancelled, normal exit
				return
			}
			c.logger.Warn("Feishu WebSocket connection error", zap.Error(err))
		}

		if !c.running {
			break
		}

		// Wait 5 seconds before reconnecting
		select {
		case <-time.After(5 * time.Second):
		case <-c.ctx.Done():
			return
		}
	}
}

// Stop stops the Feishu channel
func (c *Channel) Stop() {
	c.running = false
	if c.cancel != nil {
		c.cancel()
	}

	// WebSocket client will automatically close after context cancellation
	// Wait for background tasks to complete
	c.bgTasks.Wait()

	c.logger.Info("Feishu channel stopped")
}

// Config returns the channel configuration
func (c *Channel) Config() *Config {
	return c.config
}

// isCrossAppError checks if the error is a Feishu cross-app open_id error
func isCrossAppError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "99992361") || strings.Contains(errStr, "open_id cross app")
}
