package feishu

import (
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

// Factory creates a Feishu channel from configuration
func Factory(config map[string]interface{}, logger *zap.Logger) (*Channel, error) {
	cfg := &Config{
		AppID:             getString(config, "app_id"),
		AppSecret:         getString(config, "app_secret"),
		EncryptKey:        getString(config, "encrypt_key"),
		VerificationToken: getString(config, "verification_token"),
		ChannelCode:       getString(config, "channel_code"),
		ChannelID:         getString(config, "channel_id"),
		AgentCode:         getString(config, "agent_code"),
	}

	// Parse allow_from array
	if allowFrom, ok := config["allow_from"].([]interface{}); ok {
		cfg.AllowFrom = make([]string, 0, len(allowFrom))
		for _, item := range allowFrom {
			if s, ok := item.(string); ok {
				cfg.AllowFrom = append(cfg.AllowFrom, s)
			}
		}
	}

	// Create a new message bus for the channel
	messageBus := bus.NewMessageBus(logger)

	return NewChannel(cfg, messageBus, logger), nil
}

// getString safely extracts a string from a map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
