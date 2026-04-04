/**
 * Hook 系统接口定义
 */
package domain

import (
	"context"
	"errors"
	"fmt"
	"sort"
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
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
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

// ============================================================================
// Requirement State Change Hook
// ============================================================================

// RequirementStateEvent 需求状态变更事件
type RequirementStateEvent string

const (
	RequirementEventDispatching     RequirementStateEvent = "dispatching"
	RequirementEventDispatched     RequirementStateEvent = "dispatched"
	RequirementEventDispatchFailed RequirementStateEvent = "dispatch_failed"
	RequirementEventCodingStarted  RequirementStateEvent = "coding_started"
	RequirementEventCodingCompleted RequirementStateEvent = "coding_completed"
	RequirementEventCodingFailed   RequirementStateEvent = "coding_failed"
	RequirementEventCompleted      RequirementStateEvent = "completed"
)

// StateChange 状态变更信息
type StateChange struct {
	FromStatus RequirementStatus
	ToStatus   RequirementStatus
	Trigger    string
	Reason     string
	Timestamp  time.Time
}

// RequirementStateHook 需求状态变更钩子接口
type RequirementStateHook interface {
	Hook
	OnRequirementStateChanged(ctx context.Context, req *Requirement, change *StateChange) error
}

// RequirementStateHookRegistry 需求状态钩子注册表接口
type RequirementStateHookRegistry interface {
	Register(hook RequirementStateHook) error
	Unregister(name string) error
	Get(name string) RequirementStateHook
	List() []RequirementStateHook
	Enable(name string) error
	Disable(name string) error
}

// requirementStateHookRegistry 注册表实现
type requirementStateHookRegistry struct {
	mu    sync.RWMutex
	hooks map[string]RequirementStateHook
}

// NewRequirementStateHookRegistry 创建注册表
func NewRequirementStateHookRegistry() RequirementStateHookRegistry {
	return &requirementStateHookRegistry{
		hooks: make(map[string]RequirementStateHook),
	}
}

func (r *requirementStateHookRegistry) Register(hook RequirementStateHook) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if hook == nil {
		return errors.New("hook cannot be nil")
	}

	if _, exists := r.hooks[hook.Name()]; exists {
		return fmt.Errorf("hook %s already registered", hook.Name())
	}

	r.hooks[hook.Name()] = hook
	return nil
}

func (r *requirementStateHookRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.hooks[name]; !exists {
		return fmt.Errorf("hook %s not found", name)
	}

	delete(r.hooks, name)
	return nil
}

func (r *requirementStateHookRegistry) Get(name string) RequirementStateHook {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.hooks[name]
}

func (r *requirementStateHookRegistry) List() []RequirementStateHook {
	r.mu.RLock()
	defer r.mu.RUnlock()

	hooks := make([]RequirementStateHook, 0, len(r.hooks))
	for _, hook := range r.hooks {
		hooks = append(hooks, hook)
	}
	return hooks
}

func (r *requirementStateHookRegistry) Enable(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	hook, exists := r.hooks[name]
	if !exists {
		return fmt.Errorf("hook %s not found", name)
	}
	hook.SetEnabled(true)
	return nil
}

func (r *requirementStateHookRegistry) Disable(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	hook, exists := r.hooks[name]
	if !exists {
		return fmt.Errorf("hook %s not found", name)
	}
	hook.SetEnabled(false)
	return nil
}

// RequirementStateHookExecutor 需求状态钩子执行器
type RequirementStateHookExecutor struct {
	registry RequirementStateHookRegistry
	logger   RequirementStateHookLogger
}

// RequirementStateHookLogger 日志接口
type RequirementStateHookLogger interface {
	Debug(msg string, fields ...RequirementStateHookLogField)
	Info(msg string, fields ...RequirementStateHookLogField)
	Error(msg string, fields ...RequirementStateHookLogField)
}

// RequirementStateHookLogField 日志字段接口
type RequirementStateHookLogField interface{}

// StringField 字符串字段
type StringField struct {
	Key string
	Val string
}

// String 创建字符串日志字段
func String(key, val string) RequirementStateHookLogField {
	return StringField{Key: key, Val: val}
}

// AnyField 任意类型字段
type AnyField struct {
	Key string
	Val interface{}
}

// Any 创建任意类型日志字段
func Any(key string, val interface{}) RequirementStateHookLogField {
	return AnyField{Key: key, Val: val}
}

// NewRequirementStateHookExecutor 创建执行器
func NewRequirementStateHookExecutor(
	registry RequirementStateHookRegistry,
	logger RequirementStateHookLogger,
) *RequirementStateHookExecutor {
	return &RequirementStateHookExecutor{
		registry: registry,
		logger:   logger,
	}
}

// Execute 执行所有已注册的状态变更钩子
func (e *RequirementStateHookExecutor) Execute(ctx context.Context, req *Requirement, change *StateChange) {
	hooks := e.getEnabledHooks()
	hooks = e.sortByPriority(hooks)

	for _, hook := range hooks {
		if err := e.executeHook(ctx, hook, req, change); err != nil {
			e.logger.Error("requirement state hook execution failed",
				String("hook", hook.Name()),
				String("trigger", change.Trigger),
				Any("error", err),
			)
		}
	}
}

func (e *RequirementStateHookExecutor) executeHook(ctx context.Context, hook RequirementStateHook, req *Requirement, change *StateChange) error {
	e.logger.Debug("executing requirement state hook",
		String("hook", hook.Name()),
		String("trigger", change.Trigger),
		String("from_status", string(change.FromStatus)),
		String("to_status", string(change.ToStatus)),
	)

	return hook.OnRequirementStateChanged(ctx, req, change)
}

func (e *RequirementStateHookExecutor) getEnabledHooks() []RequirementStateHook {
	hooks := e.registry.List()
	var enabled []RequirementStateHook
	for _, hook := range hooks {
		if hook.Enabled() {
			enabled = append(enabled, hook)
		}
	}
	return enabled
}

func (e *RequirementStateHookExecutor) sortByPriority(hooks []RequirementStateHook) []RequirementStateHook {
	sort.Slice(hooks, func(i, j int) bool {
		return hooks[i].Priority() < hooks[j].Priority()
	})
	return hooks
}

// ============================================================================
// Configurable Hook System (数据库驱动)
// ============================================================================

// RequirementHookConfig Hook 配置
type RequirementHookConfig struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`     // 关联的项目ID，空表示全局Hook
	Name         string    `json:"name"`
	TriggerPoint string    `json:"trigger_point"` // start_dispatch, mark_coding, mark_failed, mark_pr_opened
	ActionType   string    `json:"action_type"`   // coding_agent, notification, webhook
	ActionConfig string    `json:"action_config"` // JSON 配置
	Enabled      bool      `json:"enabled"`
	Priority     int       `json:"priority"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// RequirementHookActionLog 执行日志
type RequirementHookActionLog struct {
	ID            string     `json:"id"`
	HookConfigID  string     `json:"hook_config_id"`
	RequirementID string     `json:"requirement_id"`
	TriggerPoint  string     `json:"trigger_point"`
	ActionType    string     `json:"action_type"`
	Status        string     `json:"status"` // pending, running, success, failed
	InputContext  string     `json:"input_context"`
	Result        string     `json:"result"`
	Error         string     `json:"error"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at"`
}

// RequirementHookConfigRepository Hook 配置仓储接口
type RequirementHookConfigRepository interface {
	Save(ctx context.Context, config *RequirementHookConfig) error
	FindByID(ctx context.Context, id string) (*RequirementHookConfig, error)
	FindByTriggerPoint(ctx context.Context, triggerPoint string) ([]*RequirementHookConfig, error)
	FindByProjectID(ctx context.Context, projectID string) ([]*RequirementHookConfig, error)
	FindEnabledByTriggerPoint(ctx context.Context, triggerPoint string) ([]*RequirementHookConfig, error)
	Delete(ctx context.Context, id string) error
}

// RequirementHookActionLogRepository 执行日志仓储接口
type RequirementHookActionLogRepository interface {
	Save(ctx context.Context, log *RequirementHookActionLog) error
	FindByRequirementID(ctx context.Context, requirementID string) ([]*RequirementHookActionLog, error)
	FindByHookConfigID(ctx context.Context, hookConfigID string, limit int) ([]*RequirementHookActionLog, error)
	FindByHookConfigAndRequirement(ctx context.Context, hookConfigID, requirementID string) (*RequirementHookActionLog, error)
}

// TriggerAgentActionConfig 触发 Agent 动作配置
type TriggerAgentActionConfig struct {
	PromptTemplate    string `json:"prompt_template"`
	TimeoutMinutes    int    `json:"timeout_minutes"`
	WorkspaceTemplate string `json:"workspace_template"`
}

// NotificationActionConfig 通知动作配置
type NotificationActionConfig struct {
	Channel  string `json:"channel"`  // feishu, email
	Template string `json:"template"`
}

// WebhookActionConfig Webhook 动作配置
type WebhookActionConfig struct {
	URL          string            `json:"url"`
	Method      string            `json:"method"`
	Headers     map[string]string `json:"headers"`
	BodyTemplate string           `json:"body_template"`
}

// ActionExecutor 动作执行器接口
type ActionExecutor interface {
	Supports(actionType string) bool
	Execute(ctx context.Context, config *RequirementHookConfig, req *Requirement, change *StateChange) (*ActionResult, error)
}

// ActionResult 动作执行结果
type ActionResult struct {
	Success bool
	Output string
	Error  error
}

// ConfigurableHookLogger 日志接口
type ConfigurableHookLogger interface {
	Debug(msg string, fields ...RequirementStateHookLogField)
	Info(msg string, fields ...RequirementStateHookLogField)
	Error(msg string, fields ...RequirementStateHookLogField)
}

// ConfigurableHookExecutor 可配置 Hook 执行器
// 从数据库加载配置，按配置执行动作
type ConfigurableHookExecutor struct {
	configRepo  RequirementHookConfigRepository
	logRepo     RequirementHookActionLogRepository
	executors   []ActionExecutor
	logger      ConfigurableHookLogger
	idGenerator IDGenerator
}

// NewConfigurableHookExecutor 创建执行器
func NewConfigurableHookExecutor(
	configRepo RequirementHookConfigRepository,
	logRepo RequirementHookActionLogRepository,
	executors []ActionExecutor,
	logger ConfigurableHookLogger,
	idGenerator IDGenerator,
) *ConfigurableHookExecutor {
	return &ConfigurableHookExecutor{
		configRepo:  configRepo,
		logRepo:     logRepo,
		executors:   executors,
		logger:      logger,
		idGenerator: idGenerator,
	}
}

// Execute 执行指定触发点的所有已配置 Hook
func (e *ConfigurableHookExecutor) Execute(
	ctx context.Context,
	triggerPoint string,
	req *Requirement,
	change *StateChange,
) {
	// 如果没有配置仓储，跳过执行
	if e.configRepo == nil {
		fmt.Printf("[DEBUG] ConfigurableHookExecutor.Execute: configRepo is NIL\n")
		return
	}

	fmt.Printf("[DEBUG] ConfigurableHookExecutor.Execute: trigger=%s, requirement=%s\n", triggerPoint, req.ID())

	// 1. 从数据库加载该触发点的配置
	configs, err := e.configRepo.FindEnabledByTriggerPoint(ctx, triggerPoint)
	if err != nil {
		e.logger.Error("failed to load hook configs",
			String("trigger", triggerPoint),
			Any("error", err),
		)
		return
	}

	if len(configs) == 0 {
		fmt.Printf("[DEBUG] No configs found for trigger: %s\n", triggerPoint)
		return
	}

	fmt.Printf("[DEBUG] Found %d configs for trigger: %s\n", len(configs), triggerPoint)

	// 2. 按优先级排序
	sorted := make([]*RequirementHookConfig, len(configs))
	copy(sorted, configs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})

	// 3. 遍历执行
	for _, config := range sorted {
		e.executeConfig(ctx, config, req, change)
	}
}

func (e *ConfigurableHookExecutor) executeConfig(
	ctx context.Context,
	config *RequirementHookConfig,
	req *Requirement,
	change *StateChange,
) {
	fmt.Printf("[DEBUG] executeConfig: configID=%s, actionType=%s\n", config.ID, config.ActionType)

	// 0. 检查是否已经执行过（避免重复触发）
	if e.logRepo != nil {
		existingLog, err := e.logRepo.FindByHookConfigAndRequirement(ctx, config.ID, req.ID().String())
		if err == nil && existingLog != nil && existingLog.Status == "success" {
			fmt.Printf("[DEBUG] Hook already executed successfully for this requirement, skipping: configID=%s, requirement=%s\n", config.ID, req.ID())
			return
		}
	}

	// 1. 创建执行日志
	logID := ""
	if e.idGenerator != nil {
		logID = e.idGenerator.Generate()
	}
	log := &RequirementHookActionLog{
		ID:            logID,
		HookConfigID:  config.ID,
		RequirementID: req.ID().String(),
		TriggerPoint:  change.Trigger,
		ActionType:    config.ActionType,
		Status:        "pending",
		StartedAt:     time.Now(),
	}
	if e.logRepo != nil {
		if err := e.logRepo.Save(ctx, log); err != nil {
			e.logger.Error("failed to save hook action log",
				String("config_id", config.ID),
				String("requirement_id", req.ID().String()),
				Any("error", err),
			)
		}
	}

	// 2. 查找对应的动作执行器
	var executor ActionExecutor
	for _, ae := range e.executors {
		if ae.Supports(config.ActionType) {
			executor = ae
			fmt.Printf("[DEBUG] Found executor for actionType: %s\n", config.ActionType)
			break
		}
	}
	if executor == nil {
		fmt.Printf("[DEBUG] No executor found for actionType: %s (registered executors: %d)\n", config.ActionType, len(e.executors))
		for i, ae := range e.executors {
			fmt.Printf("[DEBUG] Executor %d supports: %v\n", i, ae.Supports("test"))
		}
		log.Status = "failed"
		log.Error = fmt.Sprintf("no executor for action type: %s", config.ActionType)
		if e.logRepo != nil {
			_ = e.logRepo.Save(ctx, log)
		}
		return
	}

	// 3. 执行动作
	log.Status = "running"
	if e.logRepo != nil {
		_ = e.logRepo.Save(ctx, log)
	}

	result, err := executor.Execute(ctx, config, req, change)

	// 4. 更新日志
	now := time.Now()
	log.CompletedAt = &now
	if err != nil {
		log.Status = "failed"
		log.Error = err.Error()
	} else {
		log.Status = "success"
		log.Result = result.Output
	}
	if e.logRepo != nil {
		_ = e.logRepo.Save(ctx, log)
	}
}

// AddExecutor 添加动作执行器
func (e *ConfigurableHookExecutor) AddExecutor(executor ActionExecutor) {
	e.executors = append(e.executors, executor)
}
