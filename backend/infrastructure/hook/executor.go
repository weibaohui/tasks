/**
 * Hook 执行器实现
 */
package hook

import (
	"fmt"
	"sort"

	"github.com/weibh/taskmanager/domain"
	"go.uber.org/zap"
)

// ErrorStrategy 错误处理策略
type ErrorStrategy int

const (
	// ErrorStrategyStopOnFirst 遇到错误立即停止
	ErrorStrategyStopOnFirst ErrorStrategy = iota
	// ErrorStrategyContinue 继续执行后续 Hook
	ErrorStrategyContinue
)

// Executor Hook 执行器
type Executor struct {
	registry      Registry
	logger        *zap.Logger
	errorStrategy ErrorStrategy
}

// NewExecutor 创建执行器
func NewExecutor(registry Registry, logger *zap.Logger) *Executor {
	return &Executor{
		registry:      registry,
		logger:        logger,
		errorStrategy: ErrorStrategyContinue,
	}
}

// SetErrorStrategy 设置错误处理策略
func (e *Executor) SetErrorStrategy(strategy ErrorStrategy) {
	e.errorStrategy = strategy
}

// ExecutePreLLMCall 执行 PreLLMCall 钩子
func (e *Executor) ExecutePreLLMCall(ctx *domain.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
	hooks := e.getAllEnabledHooks() // 获取所有启用的 hooks
	hooks = e.sortByPriority(hooks)

	modifiedCtx := callCtx
	for _, hook := range hooks {
		llmHook, ok := hook.(domain.LLMHook)
		if !ok {
			continue
		}

		e.logger.Debug("executing PreLLMCall",
			zap.String("hook", hook.Name()),
			zap.Int("priority", hook.Priority()))

		result, err := llmHook.PreLLMCall(ctx, modifiedCtx)
		if err != nil {
			e.logger.Error("PreLLMCall failed",
				zap.String("hook", hook.Name()),
				zap.Error(err))

			ctx.AddError(err, hook.Name(), "pre_llm_call")

			if e.errorStrategy == ErrorStrategyStopOnFirst {
				return nil, fmt.Errorf("hook %s: %w", hook.Name(), err)
			}
			continue
		}

		if result != nil {
			modifiedCtx = result
		}
		ctx.AddHook(hook.Name())
	}

	return modifiedCtx, nil
}

// ExecutePostLLMCall 执行 PostLLMCall 钩子
func (e *Executor) ExecutePostLLMCall(ctx *domain.HookContext, callCtx *domain.LLMCallContext, resp *domain.LLMResponse) (*domain.LLMResponse, error) {
	hooks := e.getAllEnabledHooks()
	hooks = e.sortByPriority(hooks)

	modifiedResp := resp
	for _, hook := range hooks {
		llmHook, ok := hook.(domain.LLMHook)
		if !ok {
			continue
		}

		e.logger.Debug("executing PostLLMCall",
			zap.String("hook", hook.Name()),
			zap.Int("priority", hook.Priority()))

		result, err := llmHook.PostLLMCall(ctx, callCtx, modifiedResp)
		if err != nil {
			e.logger.Error("PostLLMCall failed",
				zap.String("hook", hook.Name()),
				zap.Error(err))

			ctx.AddError(err, hook.Name(), "post_llm_call")

			if e.errorStrategy == ErrorStrategyStopOnFirst {
				return nil, fmt.Errorf("hook %s: %w", hook.Name(), err)
			}
			continue
		}

		if result != nil {
			modifiedResp = result
		}
		ctx.AddHook(hook.Name())
	}

	return modifiedResp, nil
}

// ExecutePreToolCall 执行 PreToolCall 钩子
func (e *Executor) ExecutePreToolCall(ctx *domain.HookContext, callCtx *domain.ToolCallContext) (*domain.ToolCallContext, error) {
	hooks := e.getAllEnabledHooks()
	hooks = e.sortByPriority(hooks)

	modifiedCtx := callCtx
	for _, hook := range hooks {
		toolHook, ok := hook.(domain.ToolHook)
		if !ok {
			continue
		}

		e.logger.Debug("executing PreToolCall",
			zap.String("hook", hook.Name()),
			zap.String("tool", callCtx.ToolName))

		result, err := toolHook.PreToolCall(ctx, modifiedCtx)
		if err != nil {
			e.logger.Error("PreToolCall failed",
				zap.String("hook", hook.Name()),
				zap.Error(err))

			ctx.AddError(err, hook.Name(), "pre_tool_call")

			if e.errorStrategy == ErrorStrategyStopOnFirst {
				return nil, fmt.Errorf("hook %s: %w", hook.Name(), err)
			}
			continue
		}

		if result != nil {
			modifiedCtx = result
		}
		ctx.AddHook(hook.Name())
	}

	return modifiedCtx, nil
}

// ExecutePostToolCall 执行 PostToolCall 钩子
func (e *Executor) ExecutePostToolCall(ctx *domain.HookContext, callCtx *domain.ToolCallContext, result *domain.ToolExecutionResult) (*domain.ToolExecutionResult, error) {
	hooks := e.getAllEnabledHooks()
	hooks = e.sortByPriority(hooks)

	modifiedResult := result
	for _, hook := range hooks {
		toolHook, ok := hook.(domain.ToolHook)
		if !ok {
			continue
		}

		e.logger.Debug("executing PostToolCall",
			zap.String("hook", hook.Name()),
			zap.String("tool", callCtx.ToolName))

		res, err := toolHook.PostToolCall(ctx, callCtx, modifiedResult)
		if err != nil {
			e.logger.Error("PostToolCall failed",
				zap.String("hook", hook.Name()),
				zap.Error(err))

			ctx.AddError(err, hook.Name(), "post_tool_call")

			if e.errorStrategy == ErrorStrategyStopOnFirst {
				return nil, fmt.Errorf("hook %s: %w", hook.Name(), err)
			}
			continue
		}

		if res != nil {
			modifiedResult = res
		}
		ctx.AddHook(hook.Name())
	}

	return modifiedResult, nil
}

// ExecuteOnToolError 执行 OnToolError 钩子
func (e *Executor) ExecuteOnToolError(ctx *domain.HookContext, callCtx *domain.ToolCallContext, err error) (*domain.ToolExecutionResult, error) {
	hooks := e.getAllEnabledHooks()
	hooks = e.sortByPriority(hooks)

	modifiedResult := &domain.ToolExecutionResult{Success: false, Error: err}
	for _, hook := range hooks {
		toolHook, ok := hook.(domain.ToolHook)
		if !ok {
			continue
		}

		e.logger.Debug("executing OnToolError",
			zap.String("hook", hook.Name()),
			zap.String("tool", callCtx.ToolName))

		res, err := toolHook.OnToolError(ctx, callCtx, err)
		if err != nil {
			e.logger.Error("OnToolError failed",
				zap.String("hook", hook.Name()),
				zap.Error(err))

			ctx.AddError(err, hook.Name(), "on_tool_error")

			if e.errorStrategy == ErrorStrategyStopOnFirst {
				return nil, fmt.Errorf("hook %s: %w", hook.Name(), err)
			}
			continue
		}

		if res != nil {
			modifiedResult = res
		}
		ctx.AddHook(hook.Name())
	}

	return modifiedResult, nil
}

func (e *Executor) getEnabledHooks(hookType domain.HookType) []domain.Hook {
	hooks := e.registry.ListByType(hookType)
	var enabled []domain.Hook
	for _, hook := range hooks {
		if hook.Enabled() {
			enabled = append(enabled, hook)
		}
	}
	return enabled
}

// getAllEnabledHooks 获取所有启用的 hooks（不按类型过滤）
func (e *Executor) getAllEnabledHooks() []domain.Hook {
	hooks := e.registry.List()
	var enabled []domain.Hook
	for _, hook := range hooks {
		if hook.Enabled() {
			enabled = append(enabled, hook)
		}
	}
	return enabled
}

func (e *Executor) sortByPriority(hooks []domain.Hook) []domain.Hook {
	sorted := make([]domain.Hook, len(hooks))
	copy(sorted, hooks)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority() < sorted[j].Priority()
	})
	return sorted
}
