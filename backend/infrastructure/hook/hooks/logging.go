/**
 * 日志 Hook 实现
 */
package hooks

import (
	"github.com/weibh/taskmanager/domain"
	"go.uber.org/zap"
)

// LoggingHook 记录所有 Hook 调用日志
type LoggingHook struct {
	*domain.BaseHook
	logger *zap.Logger
}

// NewLoggingHook 创建日志 Hook
func NewLoggingHook(logger *zap.Logger) *LoggingHook {
	return &LoggingHook{
		BaseHook: domain.NewBaseHook("logging", 100, domain.HookTypeLLM),
		logger: logger,
	}
}

// PreLLMCall 记录 LLM 调用前
func (h *LoggingHook) PreLLMCall(ctx *domain.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
	h.logger.Info("PreLLMCall",
		zap.String("model", callCtx.Model),
		zap.Int("prompt_len", len(callCtx.Prompt)),
		zap.String("session_id", callCtx.SessionID),
		zap.String("trace_id", callCtx.TraceID))
	return callCtx, nil
}

// PostLLMCall 记录 LLM 调用后
func (h *LoggingHook) PostLLMCall(ctx *domain.HookContext, callCtx *domain.LLMCallContext, resp *domain.LLMResponse) (*domain.LLMResponse, error) {
	h.logger.Info("PostLLMCall",
		zap.String("model", resp.Model),
		zap.Int("content_len", len(resp.Content)),
		zap.Int("prompt_tokens", resp.Usage.PromptTokens),
		zap.Int("completion_tokens", resp.Usage.CompletionTokens),
		zap.Int("total_tokens", resp.Usage.TotalTokens),
		zap.String("finish_reason", resp.FinishReason))
	return resp, nil
}

// PreToolCall 记录工具调用前
func (h *LoggingHook) PreToolCall(ctx *domain.HookContext, callCtx *domain.ToolCallContext) (*domain.ToolCallContext, error) {
	h.logger.Info("PreToolCall",
		zap.String("tool_name", callCtx.ToolName),
		zap.Any("tool_input", callCtx.ToolInput),
		zap.String("session_id", callCtx.SessionID),
		zap.String("trace_id", callCtx.TraceID))
	return callCtx, nil
}

// PostToolCall 记录工具调用后
func (h *LoggingHook) PostToolCall(ctx *domain.HookContext, callCtx *domain.ToolCallContext, result *domain.ToolExecutionResult) (*domain.ToolExecutionResult, error) {
	h.logger.Info("PostToolCall",
		zap.String("tool_name", callCtx.ToolName),
		zap.Bool("success", result.Success),
		zap.Duration("duration", result.Duration),
		zap.Bool("cache_hit", result.CacheHit))
	return result, nil
}

// OnToolError 记录工具错误
func (h *LoggingHook) OnToolError(ctx *domain.HookContext, callCtx *domain.ToolCallContext, err error) (*domain.ToolExecutionResult, error) {
	h.logger.Error("OnToolError",
		zap.String("tool_name", callCtx.ToolName),
		zap.Error(err))
	return &domain.ToolExecutionResult{Success: false, Error: err}, nil
}
