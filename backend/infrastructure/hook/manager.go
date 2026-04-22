/**
 * Hook 管理器实现
 */
package hook

import (
	"sync"

	"github.com/weibh/taskmanager/domain"
	"go.uber.org/zap"
)

// ManagerConfig Manager 配置
type ManagerConfig struct {
	ErrorStrategy ErrorStrategy
	EnableLogging bool
}

// Manager Hook 管理器
type Manager struct {
	mu       sync.RWMutex
	registry Registry
	executor *Executor
	logger   *zap.Logger
	config   *ManagerConfig
}

// NewManager 创建 Manager
func NewManager(logger *zap.Logger, config *ManagerConfig) *Manager {
	if config == nil {
		config = &ManagerConfig{
			ErrorStrategy: ErrorStrategyContinue,
			EnableLogging: true,
		}
	}

	registry := NewRegistry()
	executor := NewExecutor(registry, logger)
	executor.SetErrorStrategy(config.ErrorStrategy)

	return &Manager{
		registry: registry,
		executor: executor,
		logger:   logger,
		config:   config,
	}
}

// Register 注册 Hook
func (m *Manager) Register(h domain.Hook) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.registry.Register(h)
}

// Unregister 注销 Hook
func (m *Manager) Unregister(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.registry.Unregister(name)
}

// List 列出所有 Hook
func (m *Manager) List() []domain.Hook {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.registry.List()
}

// Get 获取 Hook
func (m *Manager) Get(name string) domain.Hook {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.registry.Get(name)
}

// Enable 启用 Hook
func (m *Manager) Enable(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.registry.Enable(name)
}

// Disable 禁用 Hook
func (m *Manager) Disable(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.registry.Disable(name)
}

// PreLLMCall 执行 PreLLMCall 钩子
func (m *Manager) PreLLMCall(ctx *domain.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
	return m.executor.ExecutePreLLMCall(ctx, callCtx)
}

// PostLLMCall 执行 PostLLMCall 钩子
func (m *Manager) PostLLMCall(ctx *domain.HookContext, callCtx *domain.LLMCallContext, resp *domain.LLMResponse) (*domain.LLMResponse, error) {
	return m.executor.ExecutePostLLMCall(ctx, callCtx, resp)
}

// PreToolCall 执行 PreToolCall 钩子
func (m *Manager) PreToolCall(ctx *domain.HookContext, callCtx *domain.ToolCallContext) (*domain.ToolCallContext, error) {
	return m.executor.ExecutePreToolCall(ctx, callCtx)
}

// PostToolCall 执行 PostToolCall 钩子
func (m *Manager) PostToolCall(ctx *domain.HookContext, callCtx *domain.ToolCallContext, result *domain.ToolExecutionResult) (*domain.ToolExecutionResult, error) {
	return m.executor.ExecutePostToolCall(ctx, callCtx, result)
}

// OnToolError 执行 OnToolError 钩子
func (m *Manager) OnToolError(ctx *domain.HookContext, callCtx *domain.ToolCallContext, err error) (*domain.ToolExecutionResult, error) {
	return m.executor.ExecuteOnToolError(ctx, callCtx, err)
}

// OnLLMCalledWithTools 当 LLM 返回包含 tool_calls 时调用
func (m *Manager) OnLLMCalledWithTools(ctx *domain.HookContext, callCtx *domain.LLMCallContext, resp *domain.LLMResponse) {
	m.executor.ExecuteOnLLMCalledWithTools(ctx, callCtx, resp)
}

// OnToolExecutionComplete 当工具执行完成时调用
func (m *Manager) OnToolExecutionComplete(ctx *domain.HookContext) {
	m.executor.ExecuteOnToolExecutionComplete(ctx)
}

// OnThinking 当 LLM 生成思考过程时调用
func (m *Manager) OnThinking(ctx *domain.HookContext, thinking string) {
	m.executor.ExecuteOnThinking(ctx, thinking)
}
