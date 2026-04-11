package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/hook/hooks"
	"github.com/weibh/taskmanager/infrastructure/llm"
	"go.uber.org/zap"
	"github.com/weibh/taskmanager/infrastructure/trace"
)

// toolHookAdapter 将 domain.ToolHook 适配为 llm.ToolHook
type toolHookAdapter struct {
	processor    *MessageProcessor
	hookCtx      *domain.HookContext
	sessionID    string
	traceID      string
	spanID       string
	parentSpanID string
	// scope 信息
	sessionKey  string
	userCode    string
	agentCode   string
	channelCode string
	channelType string
	// 当前工具执行的 context（包含 span 信息）
	currentCtx context.Context
}

func (p *MessageProcessor) newToolHookAdapter(hookCtx *domain.HookContext, sessionID, traceID, parentSpanID, sessionKey, userCode, agentCode, channelCode, channelType string) *toolHookAdapter {
	return &toolHookAdapter{
		processor:    p,
		hookCtx:      hookCtx,
		sessionID:    sessionID,
		traceID:      traceID,
		spanID:       p.idGenerator.Generate(),
		parentSpanID: parentSpanID,
		sessionKey:   sessionKey,
		userCode:     userCode,
		agentCode:    agentCode,
		channelCode:  channelCode,
		channelType:  channelType,
	}
}

func (a *toolHookAdapter) PreToolCall(toolName string, input json.RawMessage) (json.RawMessage, error) {
	// 构建 ToolCallContext
	var args map[string]interface{}
	if err := json.Unmarshal(input, &args); err != nil {
		args = map[string]interface{}{"raw": string(input)}
	}

	callCtx := &domain.ToolCallContext{
		ToolName:     toolName,
		ToolInput:    args,
		SessionID:    a.sessionID,
		TraceID:      a.traceID,
		SpanID:       a.spanID,
		ParentSpanID: a.parentSpanID,
	}

	// 将 tool_call 的 span_id 设置到 hookCtx 的 metadata 中，供工具执行时获取
	a.hookCtx.SetMetadata("span_id", a.spanID)
	a.hookCtx.SetMetadata("parent_span_id", a.parentSpanID)

	// 将 tool_call 的 span_id 和 scope 设置到 ctx 中，供 PostToolCall 使用
	ctxWithSpan := a.hookCtx.WithValue(spanKey, a.spanID)
	ctxWithScope := ctxWithSpan.WithValue(hooks.ScopeKey, hooks.ScopeInfo{
		SessionKey:  a.sessionKey,
		UserCode:    a.userCode,
		AgentCode:   a.agentCode,
		ChannelCode: a.channelCode,
		ChannelType: a.channelType,
	})
	// 使用 trace.WithSpanID 设置 span，供工具执行时通过 trace.GetSpanID 获取
	execCtx := trace.WithSpanID(ctxWithScope, a.spanID)
	if a.parentSpanID != "" {
		execCtx = trace.WithParentSpanID(execCtx, a.parentSpanID)
	}
	// 存储当前 context，供工具执行时使用
	a.currentCtx = execCtx

	// 调用 PreToolCall hooks
	if a.processor.hookManager != nil {
		modifiedCtx, err := a.processor.hookManager.PreToolCall(ctxWithScope, callCtx)
		if err != nil {
			a.processor.logger.Error("PreToolCall hook failed", zap.Error(err))
		} else if modifiedCtx != nil {
			// 如果 hook 修改了输入，返回修改后的输入
			if modifiedCtx.ToolInput != nil {
				newInput, err := json.Marshal(modifiedCtx.ToolInput)
				if err == nil {
					return newInput, nil
				}
			}
		}
	}

	return input, nil
}

// GetCurrentCtx 获取当前工具执行的 context（包含 span 信息）
func (a *toolHookAdapter) GetCurrentCtx() context.Context {
	return a.currentCtx
}

func (a *toolHookAdapter) PostToolCall(toolName string, input json.RawMessage, output string, toolErr error) {
	// 构建 ToolCallContext
	var args map[string]interface{}
	if err := json.Unmarshal(input, &args); err != nil {
		args = map[string]interface{}{"raw": string(input)}
	}

	callCtx := &domain.ToolCallContext{
		ToolName:     toolName,
		ToolInput:    args,
		SessionID:    a.sessionID,
		TraceID:      a.traceID,
		SpanID:       a.spanID,
		ParentSpanID: a.parentSpanID,
	}

	// 构建 ToolExecutionResult
	var resultOutput interface{} = output
	if toolErr != nil {
		resultOutput = fmt.Sprintf("error: %v", toolErr)
	}
	result := &domain.ToolExecutionResult{
		Success: toolErr == nil,
		Output:  resultOutput,
		Error:   toolErr,
		SpanID:  a.spanID,
	}

	// 调用 PostToolCall hooks - 使用带有 scope 信息的 ctx
	if a.processor.hookManager != nil {
		ctxWithScope := a.hookCtx.WithValue(hooks.ScopeKey, hooks.ScopeInfo{
			SessionKey:  a.sessionKey,
			UserCode:    a.userCode,
			AgentCode:   a.agentCode,
			ChannelCode: a.channelCode,
			ChannelType: a.channelType,
		})
		_, err := a.processor.hookManager.PostToolCall(ctxWithScope, callCtx, result)
		if err != nil {
			a.processor.logger.Error("PostToolCall hook failed", zap.Error(err))
		}
	}
}

// OnLLMCalledWithTools 实现 llm.ToolExecutionObserver
func (a *toolHookAdapter) OnLLMCalledWithTools(ctx context.Context, callCtx llm.LLMCallContext) {
	if a.processor.hookManager == nil {
		return
	}

	// 转换为 domain.LLMCallContext
	domainCallCtx := &domain.LLMCallContext{
		TraceID:   a.traceID,
		SessionID: a.sessionID,
		Metadata: map[string]string{
			"session_key":  a.sessionKey,
			"user_code":    a.userCode,
			"agent_code":   a.agentCode,
			"channel_code": a.channelCode,
			"channel_type": a.channelType,
		},
	}

	domainResp := &domain.LLMResponse{
		Content: callCtx.Content,
		Usage: domain.Usage{
			PromptTokens:     callCtx.Usage.PromptTokens,
			CompletionTokens: callCtx.Usage.CompletionTokens,
			TotalTokens:      callCtx.Usage.TotalTokens,
		},
	}

	domainHookCtx := domain.NewHookContext(ctx)
	domainHookCtx.SetMetadata("trace_id", a.traceID)
	domainHookCtx.SetMetadata("session_key", a.sessionKey)

	a.processor.hookManager.OnLLMCalledWithTools(domainHookCtx, domainCallCtx, domainResp)
}

// OnToolExecutionComplete 实现 llm.ToolExecutionObserver
func (a *toolHookAdapter) OnToolExecutionComplete(ctx context.Context, tools []llm.ToolCallContext) {
	if a.processor.hookManager == nil {
		return
	}

	domainHookCtx := domain.NewHookContext(ctx)
	domainHookCtx.SetMetadata("trace_id", a.traceID)
	domainHookCtx.SetMetadata("session_key", a.sessionKey)

	a.processor.hookManager.OnToolExecutionComplete(domainHookCtx)
}


