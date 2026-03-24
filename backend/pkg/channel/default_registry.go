package channel

import (
	"github.com/weibh/taskmanager/pkg/bus"
	"github.com/weibh/taskmanager/pkg/channel/feishu"
	"go.uber.org/zap"
)

// DefaultRegistry creates a new registry with all built-in channel factories registered
func DefaultRegistry(messageBus *bus.MessageBus, logger *zap.Logger) *Registry {
	r := NewRegistry(messageBus)

	// Register Feishu channel factory
	r.Register(feishu.ChannelTypeFeishu, func(config map[string]interface{}, mb *bus.MessageBus) (Channel, error) {
		return feishu.Factory(config, mb, logger)
	})

	return r
}
