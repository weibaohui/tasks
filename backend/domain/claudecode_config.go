package domain

import (
	"encoding/json"
	"time"
)

// PermissionMode defines Claude Code permission handling
type PermissionMode string

const (
	PermissionModeDefault        PermissionMode = "default"
	PermissionModeAcceptEdits    PermissionMode = "acceptEdits"
	PermissionModePlan           PermissionMode = "plan"
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
)

// SandboxNetworkConfig 网络配置
type SandboxNetworkConfig struct {
	AllowUnixSockets    bool     `json:"allow_unix_sockets,omitempty"`
	AllowAllUnixSockets bool     `json:"allow_all_unix_sockets,omitempty"`
	AllowLocalBinding   bool     `json:"allow_local_binding,omitempty"`
	HTTPProxyPort       int      `json:"http_proxy_port,omitempty"`
	SOCKSProxyPort      int      `json:"socks_proxy_port,omitempty"`
}

// SandboxIgnoreViolations 忽略的沙箱违规
type SandboxIgnoreViolations struct {
	File    []string `json:"file,omitempty"`
	Network []string `json:"network,omitempty"`
}

// McpServerConfig MCP 服务器配置
type McpServerConfig struct {
	Command string            `json:"command,omitempty"`
	Args    []string         `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// ClaudeCodeConfig Claude Code 配置
type ClaudeCodeConfig struct {
	// === Tab 1: 基本设置 ===

	// Model 模型名称
	Model string `json:"model,omitempty"`
	// SystemPrompt 系统提示词
	SystemPrompt string `json:"system_prompt,omitempty"`
	// MaxThinkingTokens 最大思考 Token 数
	MaxThinkingTokens int `json:"max_thinking_tokens,omitempty"`
	// PermissionMode 权限模式
	PermissionMode PermissionMode `json:"permission_mode,omitempty"`
	// AllowedTools 允许的工具列表
	AllowedTools []string `json:"allowed_tools,omitempty"`
	// DisallowedTools 禁止的工具列表
	DisallowedTools []string `json:"disallowed_tools,omitempty"`
	// MaxTurns 最大对话轮次
	MaxTurns int `json:"max_turns,omitempty"`
	// Cwd 工作目录
	Cwd string `json:"cwd,omitempty"`
	// Resume 是否恢复会话
	Resume *bool `json:"resume,omitempty"`
	// Timeout 超时时间（秒），默认 120 秒
	Timeout int `json:"timeout,omitempty"`

	// === Tab 2: 高级设置 ===

	// FallbackModel 备用模型
	FallbackModel string `json:"fallback_model,omitempty"`
	// AppendSystemPrompt 追加系统提示词
	AppendSystemPrompt string `json:"append_system_prompt,omitempty"`
	// FileCheckpointing 启用文件检查点
	FileCheckpointing *bool `json:"file_checkpointing,omitempty"`
	// ContinueConversation 继续会话
	ContinueConversation *bool `json:"continue_conversation,omitempty"`
	// ForkSession Fork 会话
	ForkSession *bool `json:"fork_session,omitempty"`

	// 沙箱安全
	SandboxEnabled           *bool                 `json:"sandbox_enabled,omitempty"`
	AutoAllowBashIfSandboxed *bool                `json:"auto_allow_bash_if_sandboxed,omitempty"`
	ExcludedCommands         []string              `json:"excluded_commands,omitempty"`
	SandboxNetwork           *SandboxNetworkConfig `json:"sandbox_network,omitempty"`
	IgnoreViolations         *SandboxIgnoreViolations `json:"ignore_violations,omitempty"`

	// MCP & 插件
	McpServers  map[string]McpServerConfig `json:"mcp_servers,omitempty"`
	Plugins     []string                    `json:"plugins,omitempty"`
	LocalPlugin string                      `json:"local_plugin,omitempty"`

	// 输出 & 调试
	JSONSchema              map[string]any `json:"json_schema,omitempty"`
	IncludePartialMessages  *bool           `json:"include_partial_messages,omitempty"`
	MaxBudgetUSD           float64         `json:"max_budget_usd,omitempty"`
	DebugWriter            string          `json:"debug_writer,omitempty"`
	StderrCallback         string          `json:"stderr_callback,omitempty"`

	// 其他
	Betas       []string          `json:"betas,omitempty"`
	CLIPath     string            `json:"cli_path,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	ExtraArgs   map[string]string `json:"extra_args,omitempty"`
	Settings    string            `json:"settings,omitempty"`
	SettingSources []string        `json:"setting_sources,omitempty"`
}

// boolPtr 返回布尔值的指针
func boolPtr(b bool) *bool {
	return &b
}

// DefaultClaudeCodeConfig 返回默认配置
func DefaultClaudeCodeConfig() *ClaudeCodeConfig {
	return &ClaudeCodeConfig{
		Model:                "MiniMax-M2.7-highspeed",
		MaxThinkingTokens:    8000,
		PermissionMode:      PermissionModeDefault,
		Resume:              boolPtr(true),
		Timeout:             120,
		FileCheckpointing:   boolPtr(false),
		ContinueConversation: boolPtr(false),
		ForkSession:         boolPtr(false),
		SandboxEnabled:      boolPtr(false),
		AutoAllowBashIfSandboxed: boolPtr(false),
		IncludePartialMessages: boolPtr(false),
	}
}

// ToJSON 序列化为 JSON 字符串
func (c *ClaudeCodeConfig) ToJSON() (string, error) {
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
func (c *ClaudeCodeConfig) FromJSON(data string) error {
	if data == "" {
		return nil
	}
	return json.Unmarshal([]byte(data), c)
}

// MergeWith 将另一个配置合并到当前配置（nil 字段不覆盖）
func (c *ClaudeCodeConfig) MergeWith(other *ClaudeCodeConfig) {
	if other == nil {
		return
	}

	// 基本设置
	if other.Model != "" {
		c.Model = other.Model
	}
	if other.SystemPrompt != "" {
		c.SystemPrompt = other.SystemPrompt
	}
	if other.MaxThinkingTokens > 0 {
		c.MaxThinkingTokens = other.MaxThinkingTokens
	}
	if other.PermissionMode != "" {
		c.PermissionMode = other.PermissionMode
	}
	if len(other.AllowedTools) > 0 {
		c.AllowedTools = other.AllowedTools
	}
	if len(other.DisallowedTools) > 0 {
		c.DisallowedTools = other.DisallowedTools
	}
	if other.MaxTurns > 0 {
		c.MaxTurns = other.MaxTurns
	}
	if other.Cwd != "" {
		c.Cwd = other.Cwd
	}
	if other.Resume != nil {
		c.Resume = other.Resume
	}
	if other.Timeout > 0 {
		c.Timeout = other.Timeout
	}

	// 高级设置
	if other.FallbackModel != "" {
		c.FallbackModel = other.FallbackModel
	}
	if other.AppendSystemPrompt != "" {
		c.AppendSystemPrompt = other.AppendSystemPrompt
	}
	if other.FileCheckpointing != nil {
		c.FileCheckpointing = other.FileCheckpointing
	}
	if other.ContinueConversation != nil {
		c.ContinueConversation = other.ContinueConversation
	}
	if other.ForkSession != nil {
		c.ForkSession = other.ForkSession
	}

	// 沙箱
	if other.SandboxEnabled != nil {
		c.SandboxEnabled = other.SandboxEnabled
	}
	if other.AutoAllowBashIfSandboxed != nil {
		c.AutoAllowBashIfSandboxed = other.AutoAllowBashIfSandboxed
	}
	if len(other.ExcludedCommands) > 0 {
		c.ExcludedCommands = other.ExcludedCommands
	}
	if other.SandboxNetwork != nil {
		c.SandboxNetwork = other.SandboxNetwork
	}
	if other.IgnoreViolations != nil {
		c.IgnoreViolations = other.IgnoreViolations
	}

	// MCP & 插件
	if len(other.McpServers) > 0 {
		c.McpServers = other.McpServers
	}
	if len(other.Plugins) > 0 {
		c.Plugins = other.Plugins
	}
	if other.LocalPlugin != "" {
		c.LocalPlugin = other.LocalPlugin
	}

	// 输出 & 调试
	if len(other.JSONSchema) > 0 {
		c.JSONSchema = other.JSONSchema
	}
	if other.IncludePartialMessages != nil {
		c.IncludePartialMessages = other.IncludePartialMessages
	}
	if other.MaxBudgetUSD > 0 {
		c.MaxBudgetUSD = other.MaxBudgetUSD
	}
	if other.DebugWriter != "" {
		c.DebugWriter = other.DebugWriter
	}
	if other.StderrCallback != "" {
		c.StderrCallback = other.StderrCallback
	}

	// 其他
	if len(other.Betas) > 0 {
		c.Betas = other.Betas
	}
	if other.CLIPath != "" {
		c.CLIPath = other.CLIPath
	}
	if len(other.Env) > 0 {
		c.Env = other.Env
	}
	if len(other.ExtraArgs) > 0 {
		c.ExtraArgs = other.ExtraArgs
	}
	if other.Settings != "" {
		c.Settings = other.Settings
	}
	if len(other.SettingSources) > 0 {
		c.SettingSources = other.SettingSources
	}
}

// ClaudeCodeConfigUpdatedAt 返回配置的更新时间（用于比较）
func (c *ClaudeCodeConfig) UpdatedAt() time.Time {
	// 简化实现，实际可以使用配置内容的 hash
	return time.Now()
}