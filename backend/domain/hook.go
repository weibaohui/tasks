/**
 * Hook 系统接口定义
 */
package domain

import (
	"context"
	"sync"
	"time"
)

// ============================================================================
// 扩展 Hook 接口 (用于 LLM、Tool 等)
// ============================================================================

// Hook 基础接口
type Hook interface {
	Name() string
	Priority() int
	Enabled() bool
	SetEnabled(bool)
	HookType() HookType
}

// HookType Hook 类型
type HookType string

const (
	HookTypeLifecycle   HookType = "lifecycle"
	HookTypeLLM         HookType = "llm"
	HookTypeTool        HookType = "tool"
	HookTypeMessage     HookType = "message"
	HookTypeSkill       HookType = "skill"
	HookTypeMCP         HookType = "mcp"
	HookTypePrompt      HookType = "prompt"
	HookTypeSession     HookType = "session"
	HookTypeRequirement HookType = "requirement"
)

// BaseHook 基础实现
type BaseHook struct {
	name     string
	priority int
	enabled  bool
	hookType HookType
}

// NewBaseHook 创建 BaseHook
func NewBaseHook(name string, priority int, hookType HookType) *BaseHook {
	return &BaseHook{
		name:     name,
		priority: priority,
		enabled:  true,
		hookType: hookType,
	}
}

func (h *BaseHook) Name() string       { return h.name }
func (h *BaseHook) Priority() int      { return h.priority }
func (h *BaseHook) Enabled() bool      { return h.enabled }
func (h *BaseHook) SetEnabled(b bool)  { h.enabled = b }
func (h *BaseHook) HookType() HookType { return h.hookType }

// LLMCallContext LLM 调用上下文
type LLMCallContext struct {
	Prompt        string
	UserInput     string // 用户原始输入，不包含历史
	Model         string
	Temperature   float64
	MaxTokens     int
	StopSequences []string
	SystemPrompt  string
	SessionID     string
	TraceID       string
	Metadata      map[string]string // 用于传递 session_key, channel_code, user_code 等
}

// LLMResponse LLM 响应
type LLMResponse struct {
	Content      string
	Usage        Usage
	Model        string
	FinishReason string
	RawResponse  string
	// ContainsToolCalls 表示此响应是否包含 tool_calls（LLM 决定调用工具）
	ContainsToolCalls bool
}

// Usage token 使用量
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LLMHook LLM 钩子接口
type LLMHook interface {
	Hook
	PreLLMCall(ctx *HookContext, callCtx *LLMCallContext) (*LLMCallContext, error)
	PostLLMCall(ctx *HookContext, callCtx *LLMCallContext, response *LLMResponse) (*LLMResponse, error)
}

// ToolCallContext 工具调用上下文
type ToolCallContext struct {
	ToolName     string
	ToolInput    map[string]interface{}
	SessionID    string
	TraceID      string
	SpanID       string
	ParentSpanID string
}

// ToolExecutionResult 工具执行结果
type ToolExecutionResult struct {
	Success  bool
	Output   interface{}
	Error    error
	Duration time.Duration
	CacheHit bool
	SpanID   string
}

// ToolHook 工具钩子接口
type ToolHook interface {
	Hook
	PreToolCall(ctx *HookContext, callCtx *ToolCallContext) (*ToolCallContext, error)
	PostToolCall(ctx *HookContext, callCtx *ToolCallContext, result *ToolExecutionResult) (*ToolExecutionResult, error)
	OnToolError(ctx *HookContext, callCtx *ToolCallContext, err error) (*ToolExecutionResult, error)
}

// LLMWithToolsHook LLM 带工具调用的钩子接口
// 用于在 GenerateWithTools 内部回调，监听中间 LLM 响应和工具执行完成事件
type LLMWithToolsHook interface {
	Hook
	// OnLLMCalledWithTools 当 LLM 返回包含 tool_calls 时调用（中间响应）
	OnLLMCalledWithTools(ctx *HookContext, callCtx *LLMCallContext, resp *LLMResponse)
	// OnToolExecutionComplete 当一轮工具调用完成后调用
	// 此时可以记录最终的 llm_response，parent 应为 tool_call 的 span
	OnToolExecutionComplete(ctx *HookContext)
	// OnThinking 当 LLM 生成思考过程时调用
	OnThinking(ctx *HookContext, thinking string)
}

// ============================================================================
// HookContext 定义
// ============================================================================

// HookContext 钩子执行上下文
type HookContext struct {
	context.Context
	mu        sync.RWMutex
	values    map[interface{}]interface{}
	hooks     []string
	errors    []HookError
	metadata  map[string]string
	startTime time.Time
}

// HookError Hook 执行错误
type HookError struct {
	Err      error
	HookName string
	Phase    string
}

// NewHookContext 创建 HookContext
func NewHookContext(ctx context.Context) *HookContext {
	return &HookContext{
		Context:   ctx,
		values:    make(map[interface{}]interface{}),
		hooks:     make([]string, 0),
		errors:    make([]HookError, 0),
		metadata:  make(map[string]string),
		startTime: time.Now(),
	}
}

// WithValue 设置上下文值
func (c *HookContext) WithValue(key, val interface{}) *HookContext {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = val
	return c
}

// Get 获取上下文值
func (c *HookContext) Get(key interface{}) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.values[key]
}

// AddHook 记录已执行的 Hook
func (c *HookContext) AddHook(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hooks = append(c.hooks, name)
}

// GetHooks 获取已执行的 Hook 列表
func (c *HookContext) GetHooks() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]string, len(c.hooks))
	copy(result, c.hooks)
	return result
}

// AddError 添加错误
func (c *HookContext) AddError(err error, hookName, phase string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errors = append(c.errors, HookError{Err: err, HookName: hookName, Phase: phase})
}

// GetErrors 获取所有错误
func (c *HookContext) GetErrors() []HookError {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]HookError, len(c.errors))
	copy(result, c.errors)
	return result
}

// HasErrors 是否有错误
func (c *HookContext) HasErrors() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.errors) > 0
}

// SetMetadata 设置元数据
func (c *HookContext) SetMetadata(key, val string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metadata[key] = val
}

// GetMetadata 获取元数据
func (c *HookContext) GetMetadata(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.metadata[key]
}

// Duration 获取执行时长
func (c *HookContext) Duration() time.Duration {
	return time.Since(c.startTime)
}
