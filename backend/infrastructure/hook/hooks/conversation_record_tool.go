package hooks

import (
	"encoding/json"
	"fmt"

	"github.com/weibh/taskmanager/domain"
	"go.uber.org/zap"
)

// PreToolCall 记录工具调用
func (h *ConversationRecordHook) PreToolCall(ctx *domain.HookContext, callCtx *domain.ToolCallContext) (*domain.ToolCallContext, error) {
	if callCtx == nil {
		return callCtx, nil
	}

	traceID := callCtx.TraceID
	if traceID == "" {
		traceID = h.extractTraceID(ctx)
	}

	// 生成新的 span_id 用于工具调用
	spanID := h.idGenerator.Generate()

	// 从 ctx 获取父 span_id
	// 优先使用 ToolParentSpanKey（PostLLMCall 存储的固定父级 span），
	// 这样连续工具调用都有正确的共同父级（llm_response_with_tools），而不是互相嵌套
	parentSpanID := ""
	if p, ok := ctx.Get(ToolParentSpanKey).(string); ok && p != "" {
		parentSpanID = p
	} else if p, ok := ctx.Get(SpanKey).(string); ok {
		parentSpanID = p
	}
	if parentSpanID == "" {
		parentSpanID = callCtx.ParentSpanID // 降级：使用传入的 ParentSpanID
	}

	// 工具参数 JSON
	argsJSON, _ := json.Marshal(callCtx.ToolInput)

	record, err := h.createRecord(traceID, spanID, parentSpanID, "tool_call", "tool", fmt.Sprintf("%s(%s)", callCtx.ToolName, string(argsJSON)))
	if err != nil {
		h.logger.Error("Failed to create conversation record for tool call", zap.Error(err))
		return callCtx, nil
	}

	// 设置范围
	if scope, ok := ctx.Get(ScopeKey).(ScopeInfo); ok {
		record.SetScope(scope.SessionKey, scope.UserCode, scope.AgentCode, scope.ChannelCode, scope.ChannelType)
	}

	if err := h.repo.Save(ctx.Context, record); err != nil {
		h.logger.Error("Failed to save conversation record for tool call", zap.Error(err))
	}

	// 存储当前 span_id 供 PostToolCall 使用
	ctx.WithValue(SpanKey, spanID)

	h.logger.Debug("ConversationRecord: saved tool call",
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
		zap.String("tool_name", callCtx.ToolName))

	return callCtx, nil
}

// PostToolCall 记录工具执行结果
func (h *ConversationRecordHook) PostToolCall(ctx *domain.HookContext, callCtx *domain.ToolCallContext, result *domain.ToolExecutionResult) (*domain.ToolExecutionResult, error) {
	if callCtx == nil || result == nil {
		return result, nil
	}

	traceID := callCtx.TraceID
	if traceID == "" {
		traceID = h.extractTraceID(ctx)
	}

	// 从 ctx 获取 tool_call 的 span_id 作为 parent
	toolCallSpanID, _ := ctx.Get(SpanKey).(string)
	if toolCallSpanID == "" {
		toolCallSpanID = h.idGenerator.Generate()
	}
	// 生成新的 span_id 用于 tool_result
	spanID := h.idGenerator.Generate()

	var content string
	var eventType string
	if result.Success {
		if output, ok := result.Output.(string); ok {
			content = output
		} else {
			content = fmt.Sprintf("%v", result.Output)
		}
		eventType = "tool_result"
	} else {
		if result.Error != nil {
			content = fmt.Sprintf("error: %v", result.Error)
		} else {
			content = "unknown error"
		}
		eventType = "tool_error"
	}

	// tool_result 的 parent 是 tool_call 的 span_id
	record, err := h.createRecord(traceID, spanID, toolCallSpanID, eventType, "tool_result", content)
	if err != nil {
		h.logger.Error("Failed to create conversation record for tool result", zap.Error(err))
		return result, nil
	}

	// 设置范围 - 从 ScopeKey 获取
	if scope, ok := ctx.Get(ScopeKey).(ScopeInfo); ok {
		record.SetScope(scope.SessionKey, scope.UserCode, scope.AgentCode, scope.ChannelCode, scope.ChannelType)
	}

	if err := h.repo.Save(ctx.Context, record); err != nil {
		h.logger.Error("Failed to save conversation record for tool result", zap.Error(err))
	}

	h.logger.Debug("ConversationRecord: saved tool result",
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
		zap.String("parent_span_id", toolCallSpanID),
		zap.String("tool_name", callCtx.ToolName),
		zap.Bool("success", result.Success))

	// 更新 ctx 中的 span_id 为新的 tool_result span_id，供后续调用使用
	ctx.WithValue(SpanKey, spanID)

	return result, nil
}

// OnToolError 记录工具执行错误
func (h *ConversationRecordHook) OnToolError(ctx *domain.HookContext, callCtx *domain.ToolCallContext, err error) (*domain.ToolExecutionResult, error) {
	if callCtx == nil || err == nil {
		return &domain.ToolExecutionResult{Success: false, Error: err}, nil
	}

	traceID := callCtx.TraceID
	if traceID == "" {
		traceID = h.extractTraceID(ctx)
	}

	// 从 ctx 获取 tool_call 的 span_id 作为 parent
	toolCallSpanID, _ := ctx.Get(SpanKey).(string)
	if toolCallSpanID == "" {
		toolCallSpanID = h.idGenerator.Generate()
	}
	// 生成新的 span_id 用于 tool_error
	spanID := h.idGenerator.Generate()

	record, err := h.createRecord(traceID, spanID, toolCallSpanID, "tool_error", "tool_error", fmt.Sprintf("%s: %v", callCtx.ToolName, err))
	if err != nil {
		h.logger.Error("Failed to create conversation record for tool error", zap.Error(err))
		return &domain.ToolExecutionResult{Success: false, Error: err}, nil
	}

	// 设置范围
	if scope, ok := ctx.Get(ScopeKey).(ScopeInfo); ok {
		record.SetScope(scope.SessionKey, scope.UserCode, scope.AgentCode, scope.ChannelCode, scope.ChannelType)
	}

	if err := h.repo.Save(ctx.Context, record); err != nil {
		h.logger.Error("Failed to save conversation record for tool error", zap.Error(err))
	}

	h.logger.Debug("ConversationRecord: saved tool error",
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
		zap.String("parent_span_id", toolCallSpanID),
		zap.String("tool_name", callCtx.ToolName),
		zap.Error(err))

	// 更新 ctx 中的 span_id 为新的 tool_error span_id，供后续调用使用
	ctx.WithValue(SpanKey, spanID)

	return &domain.ToolExecutionResult{Success: false, Error: err}, nil
}
