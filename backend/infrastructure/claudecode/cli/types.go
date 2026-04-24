package cli

// StreamEvent represents a single line from claude --output-format=stream-json
type StreamEvent struct {
	Type      string          `json:"type"`
	Subtype   string          `json:"subtype,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
	Message   *AssistantMsg   `json:"message,omitempty"`
	Result    string          `json:"result,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
	Usage     *TokenUsage     `json:"usage,omitempty"`
	Error     *StreamError    `json:"error,omitempty"`
	Cwd       string          `json:"cwd,omitempty"`
	Tools     []string        `json:"tools,omitempty"`
	Model     string          `json:"model,omitempty"`
	UUID      string          `json:"uuid,omitempty"`
}

// AssistantMsg represents the assistant message in stream-json
type AssistantMsg struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Role        string          `json:"role"`
	Content     []ContentBlock  `json:"content"`
	Model       string          `json:"model"`
	StopReason  *string         `json:"stop_reason,omitempty"`
	StopSeq     *string         `json:"stop_sequence,omitempty"`
	Usage       *MsgUsage       `json:"usage,omitempty"`
	ParentID    *string         `json:"parent_tool_use_id,omitempty"`
	SessionID   string          `json:"session_id,omitempty"`
	UUID        string          `json:"uuid,omitempty"`
}

// ContentBlock represents a content block in assistant message
type ContentBlock struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	Thinking  string `json:"thinking,omitempty"`
	Name      string `json:"name,omitempty"`
	Input     any    `json:"input,omitempty"`
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   any    `json:"content,omitempty"`
	IsError   *bool  `json:"is_error,omitempty"`
}

// MsgUsage represents token usage within a message
type MsgUsage struct {
	InputTokens               int     `json:"input_tokens,omitempty"`
	OutputTokens             int     `json:"output_tokens,omitempty"`
	CacheCreationInputTokens int     `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int     `json:"cache_read_input_tokens,omitempty"`
}

// TokenUsage represents final token usage from result event
type TokenUsage struct {
	InputTokens               int     `json:"input_tokens"`
	OutputTokens             int     `json:"output_tokens"`
	CacheCreationInputTokens int     `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int     `json:"cache_read_input_tokens,omitempty"`
	Total                     int     `json:"total,omitempty"`
}

// StreamError represents an error in stream
type StreamError struct {
	Type string         `json:"type,omitempty"`
	Name string         `json:"name,omitempty"`
	Data map[string]any `json:"data,omitempty"`
}

// ModelUsageEntry represents per-model usage in result
type ModelUsageEntry struct {
	InputTokens              int     `json:"inputTokens"`
	OutputTokens             int     `json:"outputTokens"`
	CacheReadInputTokens     int     `json:"cacheReadInputTokens"`
	CacheCreationInputTokens int     `json:"cacheCreationInputTokens"`
	WebSearchRequests        int     `json:"webSearchRequests"`
	CostUSD                  float64 `json:"costUSD"`
	ContextWindow            int     `json:"contextWindow"`
	MaxOutputTokens          int     `json:"maxOutputTokens"`
}
