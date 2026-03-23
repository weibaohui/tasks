package channel

import (
	"context"
	"fmt"
	"sync"

	"github.com/weibh/taskmanager/pkg/bus"
)

// ChannelType defines the type of channel
type ChannelType string

const (
	ChannelTypeFeishu    ChannelType = "feishu"
	ChannelTypeDingTalk  ChannelType = "dingtalk"
	ChannelTypeWeChat    ChannelType = "wechat"
	ChannelTypeWebSocket ChannelType = "websocket"
)

// Channel is the interface that all channel implementations must satisfy
type Channel interface {
	// Name returns the channel name
	Name() string
	// Type returns the channel type (e.g., "feishu", "dingtalk", "wechat")
	Type() string
	// Start starts the channel
	Start(ctx context.Context) error
	// Stop stops the channel
	Stop()
}

// ChannelFactory is a function that creates a channel instance
type ChannelFactory func(config map[string]interface{}) (Channel, error)

// Registry manages channel factories for creating channel instances
type Registry struct {
	factories map[string]ChannelFactory
	mu        sync.RWMutex
}

// NewRegistry creates a new channel registry
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]ChannelFactory),
	}
}

// Register registers a channel factory for a channel type
func (r *Registry) Register(channelType string, factory ChannelFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[channelType] = factory
}

// GetFactory returns the factory for a channel type
func (r *Registry) GetFactory(channelType string) (ChannelFactory, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	factory, ok := r.factories[channelType]
	return factory, ok
}

// ListTypes returns all registered channel types
func (r *Registry) ListTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	types := make([]string, 0, len(r.factories))
	for t := range r.factories {
		types = append(types, t)
	}
	return types
}

// Manager manages multiple channels
type Manager struct {
	channels map[string]Channel
	mu       sync.RWMutex
	bus      *bus.MessageBus
}

// NewManager creates a new channel manager
func NewManager(messageBus *bus.MessageBus) *Manager {
	return &Manager{
		channels: make(map[string]Channel),
		bus:      messageBus,
	}
}

// Register registers a channel
func (m *Manager) Register(ch Channel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels[ch.Name()] = ch
}

// Get retrieves a channel by name
func (m *Manager) Get(name string) Channel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.channels[name]
}

// StartAll starts all registered channels
func (m *Manager) StartAll(ctx context.Context) error {
	m.mu.RLock()
	channels := make([]Channel, 0, len(m.channels))
	for _, ch := range m.channels {
		channels = append(channels, ch)
	}
	m.mu.RUnlock()

	for _, ch := range channels {
		if err := ch.Start(ctx); err != nil {
			return fmt.Errorf("start channel %s: %w", ch.Name(), err)
		}
	}
	return nil
}

// StopAll stops all registered channels
func (m *Manager) StopAll() {
	m.mu.RLock()
	channels := make([]Channel, 0, len(m.channels))
	for _, ch := range m.channels {
		channels = append(channels, ch)
	}
	m.mu.RUnlock()

	for _, ch := range channels {
		ch.Stop()
	}
}

// List returns all channel names
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var names []string
	for name := range m.channels {
		names = append(names, name)
	}
	return names
}
