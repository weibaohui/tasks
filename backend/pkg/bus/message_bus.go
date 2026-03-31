package bus

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
)

// OutboundCallback is the callback function type for outbound messages
type OutboundCallback func(msg *OutboundMessage) error

// StreamCallback is the callback function type for streaming messages
type StreamCallback func(chunk *StreamChunk) error

// MessageBus is an async message bus that decouples channels from the agent core
type MessageBus struct {
	inbound             chan *InboundMessage
	outbound            chan *OutboundMessage
	stream              chan *StreamChunk
	outboundSubscribers map[string][]OutboundCallback
	streamSubscribers   map[string][]StreamCallback
	mu                  sync.RWMutex
	running             atomic.Bool
	logger              *zap.Logger
}

// NewMessageBus creates a new message bus
func NewMessageBus(logger *zap.Logger) *MessageBus {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &MessageBus{
		inbound:             make(chan *InboundMessage, 100),
		outbound:            make(chan *OutboundMessage, 100),
		stream:              make(chan *StreamChunk, 1000), // Streaming needs larger buffer
		outboundSubscribers: make(map[string][]OutboundCallback),
		streamSubscribers:   make(map[string][]StreamCallback),
		logger:              logger,
	}
}

// PublishInbound publishes a message from channel to agent
func (b *MessageBus) PublishInbound(msg *InboundMessage) {
	b.inbound <- msg
}

// ConsumeInbound consumes the next inbound message (blocks until available)
func (b *MessageBus) ConsumeInbound(ctx context.Context) (*InboundMessage, error) {
	select {
	case msg := <-b.inbound:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// PublishOutbound publishes a response from agent to channel
func (b *MessageBus) PublishOutbound(msg *OutboundMessage) {
	b.outbound <- msg
}

// ConsumeOutbound consumes the next outbound message (blocks until available)
func (b *MessageBus) ConsumeOutbound(ctx context.Context) (*OutboundMessage, error) {
	select {
	case msg := <-b.outbound:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// PublishStream publishes a streaming message chunk
func (b *MessageBus) PublishStream(chunk *StreamChunk) {
	select {
	case b.stream <- chunk:
	default:
		b.logger.Warn("Stream channel full, dropping chunk",
			zap.String("channel", chunk.Channel),
			zap.String("chat_id", chunk.ChatID),
		)
	}
}

// SubscribeOutbound subscribes to outbound messages for a specific channel
func (b *MessageBus) SubscribeOutbound(channel string, callback OutboundCallback) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.outboundSubscribers[channel] = append(b.outboundSubscribers[channel], callback)
}

// SubscribeStream subscribes to streaming messages for a specific channel
func (b *MessageBus) SubscribeStream(channel string, callback StreamCallback) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.streamSubscribers[channel] = append(b.streamSubscribers[channel], callback)
}

// StartDispatcher starts the outbound message dispatcher
func (b *MessageBus) StartDispatcher(ctx context.Context) {
	b.running.Store(true)
	go b.dispatchLoop(ctx)
	go b.streamDispatchLoop(ctx)
}

// dispatchLoop dispatches outbound messages to subscribed channels
func (b *MessageBus) dispatchLoop(ctx context.Context) {
	for b.running.Load() {
		select {
		case msg := <-b.outbound:
			b.dispatchToSubscribers(msg)
		case <-ctx.Done():
			b.running.Store(false)
			return
		}
	}
}

// streamDispatchLoop dispatches streaming messages to subscribed channels
func (b *MessageBus) streamDispatchLoop(ctx context.Context) {
	for b.running.Load() {
		select {
		case chunk := <-b.stream:
			b.dispatchStreamToSubscribers(chunk)
		case <-ctx.Done():
			b.running.Store(false)
			return
		}
	}
}

// dispatchToSubscribers dispatches a message to subscribers
func (b *MessageBus) dispatchToSubscribers(msg *OutboundMessage) {
	b.mu.RLock()
	subscribers := b.outboundSubscribers[msg.Channel]
	b.mu.RUnlock()

	for _, callback := range subscribers {
		if err := callback(msg); err != nil {
			// Feishu cross-app open_id error is common and noisy - log at debug level
			if isFeishuCrossAppError(err) {
				b.logger.Debug("Feishu dispatch skipped (cross-app open_id)",
					zap.String("channel", msg.Channel),
					zap.Error(err),
				)
			} else {
				b.logger.Error("Failed to dispatch message to channel",
					zap.String("channel", msg.Channel),
					zap.Error(err),
				)
			}
		}
	}
}

// isFeishuCrossAppError checks if the error is a Feishu cross-app open_id error
func isFeishuCrossAppError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "99992361") || strings.Contains(errStr, "open_id cross app")
}

// dispatchStreamToSubscribers dispatches a streaming message to subscribers
func (b *MessageBus) dispatchStreamToSubscribers(chunk *StreamChunk) {
	b.mu.RLock()
	subscribers := b.streamSubscribers[chunk.Channel]
	b.mu.RUnlock()

	for _, callback := range subscribers {
		if err := callback(chunk); err != nil {
			b.logger.Error("Failed to dispatch stream message to channel",
				zap.String("channel", chunk.Channel),
				zap.Error(err),
			)
		}
	}
}

// Stop stops the dispatcher loops
func (b *MessageBus) Stop() {
	b.running.Store(false)
}

// InboundSize returns the number of pending inbound messages
func (b *MessageBus) InboundSize() int {
	return len(b.inbound)
}

// OutboundSize returns the number of pending outbound messages
func (b *MessageBus) OutboundSize() int {
	return len(b.outbound)
}

// StreamSize returns the number of pending streaming messages
func (b *MessageBus) StreamSize() int {
	return len(b.stream)
}
