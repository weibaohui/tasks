package channel

import (
	"github.com/weibh/taskmanager/pkg/channel/feishu"
	"go.uber.org/zap"
)

// DefaultRegistry creates a new registry with all built-in channel factories registered
func DefaultRegistry(logger *zap.Logger) *Registry {
	r := NewRegistry()

	// Register Feishu channel factory
	r.Register(feishu.ChannelTypeFeishu, func(config map[string]interface{}) (Channel, error) {
		return feishu.Factory(config, logger)
	})

	return r
}
