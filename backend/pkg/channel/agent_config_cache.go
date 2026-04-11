package channel

type AgentConfigCache struct {
	cache map[string]*AgentConfig
}

func NewAgentConfigCache() *AgentConfigCache {
	return &AgentConfigCache{
		cache: make(map[string]*AgentConfig),
	}
}

// AgentConfig Agent 配置
type AgentConfig struct {
	AgentCode    string
	Name         string
	Instructions string
	Tools        []string
	MCPs         []string
}

// Get 获取配置
func (c *AgentConfigCache) Get(key string) (*AgentConfig, bool) {
	cfg, ok := c.cache[key]
	return cfg, ok
}

// Set 设置配置
func (c *AgentConfigCache) Set(key string, cfg *AgentConfig) {
	c.cache[key] = cfg
}

// Clear 清除缓存
func (c *AgentConfigCache) Clear(key string) {
	delete(c.cache, key)
}
