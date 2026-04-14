package hook

import (
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/hook/hooks"
	"go.uber.org/zap"
)

// ToolHookBridge provides a reusable bridge between CLI tool execution
// and the domain hook system. It is used by both ClaudeCodeProcessor
// and OpenCodeProcessor to avoid duplicating PreToolCall/PostToolCall logic.
type ToolHookBridge struct {
	Manager     *Manager
	Logger      *zap.Logger
	HookCtx     *domain.HookContext
	SessionKey  string
	UserCode    string
	AgentCode   string
	ChannelCode string
	ChannelType string
	TraceID     string
}

// PreToolCall executes PreToolCall hooks for the given tool.
// It also ensures ToolParentSpanKey is initialized when needed,
// which is required by conversation record hooks.
func (b *ToolHookBridge) PreToolCall(toolName string, toolInput map[string]any) error {
	if b.Manager == nil {
		return nil
	}

	// Ensure ToolParentSpanKey is set so that consecutive tool calls
	// share the same parent span instead of nesting.
	if b.HookCtx != nil && b.HookCtx.Get(hooks.ToolParentSpanKey) == nil {
		if currentSpan, ok := b.HookCtx.Get(hooks.SpanKey).(string); ok && currentSpan != "" {
			b.HookCtx.WithValue(hooks.ToolParentSpanKey, currentSpan)
		}
	}

	callCtx := &domain.ToolCallContext{
		ToolName:     toolName,
		ToolInput:    toolInput,
		SessionID:    b.SessionKey,
		TraceID:      b.TraceID,
		SpanID:       "",
		ParentSpanID: "",
	}

	_, err := b.Manager.PreToolCall(b.HookCtx, callCtx)
	if err != nil {
		if b.Logger != nil {
			b.Logger.Error("PreToolCall hook failed",
				zap.String("tool", toolName),
				zap.Error(err))
		}
	}
	return nil
}

// PostToolCall executes PostToolCall hooks for the given tool result.
func (b *ToolHookBridge) PostToolCall(toolName string, toolInput map[string]any, output string, success bool) error {
	if b.Manager == nil {
		return nil
	}

	callCtx := &domain.ToolCallContext{
		ToolName:     toolName,
		ToolInput:    toolInput,
		SessionID:    b.SessionKey,
		TraceID:      b.TraceID,
		SpanID:       "",
		ParentSpanID: "",
	}

	execResult := &domain.ToolExecutionResult{
		Success:  success,
		Duration: 0,
		Output:   output,
	}

	_, err := b.Manager.PostToolCall(b.HookCtx, callCtx, execResult)
	if err != nil {
		if b.Logger != nil {
			b.Logger.Error("PostToolCall hook failed",
				zap.String("tool", toolName),
				zap.Error(err))
		}
	}
	return nil
}
