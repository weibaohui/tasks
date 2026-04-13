package opencode

// StreamingCallback 回调接口用于流式输出（与 ClaudeCodeProcessor 接口兼容）
type StreamingCallback interface {
	OnThinking(thinking string)
	OnToolCall(toolName string, input map[string]any)
	OnToolResult(toolName string, result string)
	OnText(text string)
	OnComplete(finalResult string)
	GetFinalResult() string
}

// OpenCodeEvent OpenCode CLI JSON 事件
type OpenCodeEvent struct {
	Type      string    `json:"type"`
	Timestamp int64     `json:"timestamp"`
	SessionID string    `json:"sessionID"`
	Part      EventPart `json:"part"`
}

// EventPart 事件内容
type EventPart struct {
	ID        string     `json:"id"`
	MessageID string     `json:"messageID"`
	SessionID string     `json:"sessionID"`
	Type      string     `json:"type"`
	Text      string     `json:"text,omitempty"`
	Thinking  string     `json:"thinking,omitempty"`
	Tool      string     `json:"tool,omitempty"`
	CallID    string     `json:"callID,omitempty"`
	State     ToolState  `json:"state,omitempty"`
	Reason    string     `json:"reason,omitempty"`
	Tokens    TokenUsage `json:"tokens,omitempty"`
	Cost      float64    `json:"cost,omitempty"`
}

// ToolState 工具调用状态
type ToolState struct {
	Status  string         `json:"status"` // completed | error | pending
	Input   map[string]any `json:"input,omitempty"`
	Output  string         `json:"output,omitempty"`
	Error   *string        `json:"error,omitempty"`
}

// TokenUsage Token 使用量
type TokenUsage struct {
	Total     int `json:"total"`
	Input     int `json:"input"`
	Output    int `json:"output"`
	Reasoning int `json:"reasoning"`
	Cache     CacheUsage `json:"cache"`
}

// CacheUsage 缓存使用量
type CacheUsage struct {
	Write int `json:"write"`
	Read  int `json:"read"`
}
