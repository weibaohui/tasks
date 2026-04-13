package domain

import (
	"encoding/json"
	"time"
)

// OpenCodeAgentType 定义 OpenCode Agent 类型
type OpenCodeAgentType string

const (
	OpenCodeAgentTypeBuild      OpenCodeAgentType = "build"       // 全功能构建 Agent
	OpenCodeAgentTypePlan      OpenCodeAgentType = "plan"       // 规划 Agent（只读+计划文件）
	OpenCodeAgentTypeExplore   OpenCodeAgentType = "explore"    // 探索 Agent（只读）
	OpenCodeAgentTypeGeneral   OpenCodeAgentType = "general"    // 通用 Agent
	OpenCodeAgentTypeCompaction OpenCodeAgentType = "compaction" // 精简版 Agent
)

// OpenCodeConfig OpenCode CLI 配置
type OpenCodeConfig struct {
	// === 基本设置 ===

	// Model 模型名称，格式为 provider/model，如 openrouter/anthropic/claude-sonnet-4
	Model string `json:"model,omitempty"`
	// AgentType Agent 类型：build | plan | explore | general | compaction
	AgentType OpenCodeAgentType `json:"agent_type,omitempty"`
	// WorkDir 工作目录
	WorkDir string `json:"work_dir,omitempty"`
	// Timeout 超时时间（秒），默认 600 秒
	Timeout int `json:"timeout,omitempty"`
	// Continue 是否继续上次会话
	Continue bool `json:"continue,omitempty"`
	// SessionID 指定要继续的会话 ID
	SessionID string `json:"session_id,omitempty"`
	// Fork 是否分叉会话
	Fork bool `json:"fork,omitempty"`

	// === 高级设置 ===

	// SkipPermissions 是否跳过权限确认
	SkipPermissions bool `json:"skip_permissions,omitempty"`
	// ShowThinking 是否显示思考过程
	ShowThinking bool `json:"show_thinking,omitempty"`
	// ShareSession 是否分享会话
	ShareSession bool `json:"share_session,omitempty"`
	// Variant 模型变体（provider-specific reasoning effort）
	Variant string `json:"variant,omitempty"`

	// === 系统提示词 ===

	// SystemPrompt 系统提示词（通过环境变量传递）
	SystemPrompt string `json:"system_prompt,omitempty"`

	// === 环境配置 ===

	// Env 环境变量，会传递给 OpenCode CLI
	Env map[string]string `json:"env,omitempty"`
}

// DefaultOpenCodeConfig 返回默认配置
func DefaultOpenCodeConfig() *OpenCodeConfig {
	return &OpenCodeConfig{
		AgentType:      OpenCodeAgentTypeBuild,
		Continue:       false,
		SkipPermissions: false,
		ShowThinking:   false,
		ShareSession:   false,
	}
}

// ToJSON 序列化为 JSON 字符串
func (c *OpenCodeConfig) ToJSON() (string, error) {
	if c == nil {
		return "", nil
	}
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON 从 JSON 字符串反序列化
func (c *OpenCodeConfig) FromJSON(data string) error {
	if data == "" {
		return nil
	}
	return json.Unmarshal([]byte(data), c)
}

// MergeWith 将另一个配置合并到当前配置（nil 字段不覆盖）
func (c *OpenCodeConfig) MergeWith(other *OpenCodeConfig) {
	if other == nil {
		return
	}

	// 基本设置
	if other.Model != "" {
		c.Model = other.Model
	}
	if other.AgentType != "" {
		c.AgentType = other.AgentType
	}
	if other.WorkDir != "" {
		c.WorkDir = other.WorkDir
	}
	if other.SessionID != "" {
		c.SessionID = other.SessionID
	}

	// 高级设置
	if other.Continue {
		c.Continue = other.Continue
	}
	if other.Fork {
		c.Fork = other.Fork
	}
	if other.SkipPermissions {
		c.SkipPermissions = other.SkipPermissions
	}
	if other.ShowThinking {
		c.ShowThinking = other.ShowThinking
	}
	if other.ShareSession {
		c.ShareSession = other.ShareSession
	}
	if other.Variant != "" {
		c.Variant = other.Variant
	}

	// 系统提示词
	if other.SystemPrompt != "" {
		c.SystemPrompt = other.SystemPrompt
	}

	// 环境变量
	if len(other.Env) > 0 {
		c.Env = other.Env
	}
}

// Clone 返回配置的深拷贝
func (c *OpenCodeConfig) Clone() *OpenCodeConfig {
	if c == nil {
		return nil
	}
	data, _ := json.Marshal(c)
	clone := &OpenCodeConfig{}
	_ = json.Unmarshal(data, clone)
	return clone
}

// UpdatedAt 返回配置的更新时间
func (c *OpenCodeConfig) UpdatedAt() time.Time {
	return time.Now()
}
