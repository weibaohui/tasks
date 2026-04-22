package hooks

import (
	"context"
	"fmt"
	"strings"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/trace"
	"go.uber.org/zap"
)

// PreLLMCall 记录 LLM 调用前的用户输入
func (h *ConversationRecordHook) PreLLMCall(ctx *domain.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
	if callCtx == nil {
		return nil, nil
	}

	// 从 trace context 提取 span 信息
	spanID := trace.GetSpanID(ctx.Context)
	parentSpanID := trace.GetParentSpanID(ctx.Context)
	if spanID == "" {
		spanID = h.idGenerator.Generate()
	}

	// 把 scope 存回 callCtx.Metadata，这样 PostLLMCall 能通过 extractScope 拿到
	if callCtx.Metadata == nil {
		callCtx.Metadata = make(map[string]string)
	}
	// 先设置 session_key（从 callCtx.SessionID 获取），其他字段从 extractScope 获取
	if callCtx.SessionID != "" {
		callCtx.Metadata["session_key"] = callCtx.SessionID
	}

	// 提取范围信息（在设置 Metadata 之后，以便 extractScope 能正确获取 session_key）
	scope := h.extractScope(ctx, callCtx)
	if scope.SessionKey != "" {
		callCtx.Metadata["session_key"] = scope.SessionKey
	}
	if scope.UserCode != "" {
		callCtx.Metadata["user_code"] = scope.UserCode
	}
	if scope.AgentCode != "" {
		callCtx.Metadata["agent_code"] = scope.AgentCode
	}
	if scope.ChannelCode != "" {
		callCtx.Metadata["channel_code"] = scope.ChannelCode
	}
	if scope.ChannelType != "" {
		callCtx.Metadata["channel_type"] = scope.ChannelType
	}

	// 重新提取 scope，确保使用完整的 Metadata
	scope = h.extractScope(ctx, callCtx)
	ctx.WithValue(ScopeKey, scope)
	ctx.WithValue(SpanKey, spanID)
	ctx.WithValue(promptKey, callCtx.Prompt)

	// 记录用户输入（使用 UserInput 原始输入，不使用包含历史的 Prompt）
	userInput := callCtx.UserInput
	if userInput == "" {
		userInput = callCtx.Prompt // 降级：如果没有 UserInput 则使用完整 prompt
	}
	record, err := h.createRecord(callCtx.TraceID, spanID, parentSpanID, "llm_call", "user", userInput)
	if err != nil {
		h.logger.Error("Failed to create conversation record for user input", zap.Error(err))
		return callCtx, nil
	}

	record.SetScope(scope.SessionKey, scope.UserCode, scope.AgentCode, scope.ChannelCode, scope.ChannelType)
	if err := h.repo.Save(ctx.Context, record); err != nil {
		h.logger.Error("Failed to save conversation record for user input", zap.Error(err))
	}

	h.logger.Debug("ConversationRecord: saved user prompt",
		zap.String("trace_id", callCtx.TraceID),
		zap.String("span_id", spanID),
		zap.String("parent_span_id", parentSpanID),
		zap.Int("prompt_len", len(callCtx.Prompt)))

	return callCtx, nil
}

// PostLLMCall 记录 LLM 响应
func (h *ConversationRecordHook) PostLLMCall(ctx *domain.HookContext, callCtx *domain.LLMCallContext, resp *domain.LLMResponse) (*domain.LLMResponse, error) {
	if resp == nil {
		return resp, nil
	}

	// 获取用户输入的 span_id 作为 parent（来自 PreLLMCall 存储的 SpanKey）
	parentSpanID, _ := ctx.Get(SpanKey).(string)
	if parentSpanID == "" {
		parentSpanID = trace.GetSpanID(ctx.Context)
	}

	// 检查 resp.RawResponse 是否包含 tool_calls（LLM 决定调用工具的标志）
	hasToolCalls := containsToolCalls(resp.RawResponse)

	// LLM 响应生成新的 span_id，parent 指向上游用户输入
	spanID := h.idGenerator.Generate()

	// 存储新的 span_id 到 HookContext.values，供后续 PreToolCall 使用
	ctx.WithValue(SpanKey, spanID)

	traceID := callCtx.TraceID
	if traceID == "" {
		traceID = h.extractTraceID(ctx)
	}
	if traceID == "" {
		traceID = trace.MustGetTraceID(ctx.Context)
	}

	// 直接从 callCtx 获取 scope 信息（PreLLMCall 会设置 callCtx.Metadata）
	sessionKey := callCtx.SessionID
	userCode := ""
	agentCode := ""
	channelCode := ""
	channelType := ""
	if callCtx.Metadata != nil {
		if v := callCtx.Metadata["session_key"]; v != "" {
			sessionKey = v
		}
		userCode = callCtx.Metadata["user_code"]
		agentCode = callCtx.Metadata["agent_code"]
		channelCode = callCtx.Metadata["channel_code"]
		channelType = callCtx.Metadata["channel_type"]
	}

	// 如果 LLM 决定调用工具
	if hasToolCalls {
		toolCallsSpanID := h.idGenerator.Generate()

		// 记录工具决策：LLM 决定调用哪些工具
		if toolNames := extractToolNames(resp.RawResponse); len(toolNames) > 0 {
			decisionContent := fmt.Sprintf("调用工具: %s", strings.Join(toolNames, ", "))
			decisionRecord, err := h.createRecord(traceID, h.idGenerator.Generate(), parentSpanID, "tool_decision", "assistant", decisionContent)
			if err != nil {
				h.logger.Error("Failed to create conversation record for tool decision", zap.Error(err))
			} else {
				decisionRecord.SetScope(sessionKey, userCode, agentCode, channelCode, channelType)
				if err := h.repo.Save(context.Background(), decisionRecord); err != nil {
					h.logger.Error("Failed to save conversation record for tool decision", zap.Error(err))
				} else {
					h.logger.Debug("ConversationRecord: saved tool decision",
						zap.String("trace_id", traceID),
						zap.Strings("tool_names", toolNames))
				}
			}
		}

		// 仅当有内容时记录中间响应
		if resp.Content != "" {
			// 中间响应的 parent 是用户输入的 span
			toolCallsRecord, err := h.createRecord(traceID, toolCallsSpanID, parentSpanID, "llm_response_with_tools", "assistant", resp.Content)
			if err != nil {
				h.logger.Error("Failed to create conversation record for LLM response with tools", zap.Error(err))
			} else {
				toolCallsRecord.SetScope(sessionKey, userCode, agentCode, channelCode, channelType)
				toolCallsRecord.SetTokenUsage(resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens, 0, 0)
				if err := h.repo.Save(context.Background(), toolCallsRecord); err != nil {
					h.logger.Error("Failed to save conversation record for LLM response with tools", zap.Error(err))
				}
				h.logger.Debug("ConversationRecord: saved LLM response with tools",
					zap.String("trace_id", traceID),
					zap.String("span_id", toolCallsSpanID),
					zap.String("parent_span_id", parentSpanID))
			}
		}

		// 更新 span 链：后续的 tool_call 应以这个中间响应为 parent
		// ToolParentSpanKey 存储固定的工具调用父级，不随每次 tool 调用更新
		// SpanKey 仍更新为 toolCallsSpanID，供非工具场景使用
		// 即使 resp.Content 为空也必须设置，否则连续工具调用的 parent 会断裂
		ctx.WithValue(ToolParentSpanKey, toolCallsSpanID)
		ctx.WithValue(SpanKey, toolCallsSpanID)

		// 延迟记录最终的 llm_response：存储信息到 context，由 OnToolExecutionComplete 记录
		ctx.WithValue(deferredResponseKey, &deferredLLMResponse{
			TraceID:      traceID,
			SpanID:       spanID,
			ParentSpanID: "", // 将在 OnToolExecutionComplete 时设置为 tool_result 的 span
			Content:      resp.Content,
			Usage:        resp.Usage,
			Scope: ScopeInfo{
				SessionKey:  sessionKey,
				UserCode:    userCode,
				AgentCode:   agentCode,
				ChannelCode: channelCode,
				ChannelType: channelType,
			},
		})

		return resp, nil
	}

	// 没有 tool_calls，正常记录助手响应
	role := "assistant"
	content := resp.Content

	record, err := h.createRecord(traceID, spanID, parentSpanID, "llm_response", role, content)
	if err != nil {
		h.logger.Error("Failed to create conversation record for LLM response", zap.Error(err))
		return resp, nil
	}

	// 设置范围（直接从 callCtx 获取，PreLLMCall 会设置 callCtx.Metadata）
	record.SetScope(sessionKey, userCode, agentCode, channelCode, channelType)

	// 设置 token 使用量
	record.SetTokenUsage(resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens, 0, 0)

	if err := h.repo.Save(ctx.Context, record); err != nil {
		h.logger.Error("Failed to save conversation record for LLM response", zap.Error(err))
	}

	h.logger.Debug("ConversationRecord: saved LLM response",
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
		zap.String("role", role),
		zap.Int("content_len", len(content)))

	return resp, nil
}

// OnLLMCalledWithTools 当 LLM 返回包含 tool_calls 时调用
// 此时应该记录 llm_response_with_tools
func (h *ConversationRecordHook) OnLLMCalledWithTools(ctx *domain.HookContext, callCtx *domain.LLMCallContext, resp *domain.LLMResponse) {
	// 这个方法在 GenerateWithTools 内部调用时，工具还没有执行
	// 实际的 llm_response_with_tools 记录已经在 PostLLMCall 中处理了
	// 这里只做日志记录
	h.logger.Debug("ConversationRecord: OnLLMCalledWithTools",
		zap.String("content", resp.Content),
		zap.Int("content_len", len(resp.Content)))
}

// OnToolExecutionComplete 当一轮工具调用完成后调用
// 此时应该记录最终的 llm_response，parent 应为 llm_response_with_tools 的 span
func (h *ConversationRecordHook) OnToolExecutionComplete(ctx *domain.HookContext) {
	// 获取延迟的 LLM 响应信息
	deferredResp, ok := ctx.Get(deferredResponseKey).(*deferredLLMResponse)
	if !ok || deferredResp == nil {
		return
	}

	// 优先使用 ToolParentSpanKey（固定的 llm_response_with_tools span），
	// 这样连续工具调用的结果和最终 llm_response 都以同一个 span 为 parent，
	// 而不是互相嵌套（SpanKey 会被 PostToolCall 更新为最后一个 tool_result 的 span）
	parentSpanID, _ := ctx.Get(ToolParentSpanKey).(string)
	if parentSpanID == "" {
		parentSpanID, _ = ctx.Get(SpanKey).(string)
	}
	if parentSpanID == "" {
		h.logger.Warn("OnToolExecutionComplete: no parent span found, skipping deferred LLM response")
		return
	}

	// 记录最终的 llm_response，parent 是 tool_call 的 span
	record, err := h.createRecord(deferredResp.TraceID, deferredResp.SpanID, parentSpanID, "llm_response", "assistant", deferredResp.Content)
	if err != nil {
		h.logger.Error("Failed to create conversation record for deferred LLM response", zap.Error(err))
		return
	}

	record.SetScope(deferredResp.Scope.SessionKey, deferredResp.Scope.UserCode, deferredResp.Scope.AgentCode, deferredResp.Scope.ChannelCode, deferredResp.Scope.ChannelType)
	record.SetTokenUsage(deferredResp.Usage.PromptTokens, deferredResp.Usage.CompletionTokens, deferredResp.Usage.TotalTokens, 0, 0)

	if err := h.repo.Save(ctx.Context, record); err != nil {
		h.logger.Error("Failed to save conversation record for deferred LLM response", zap.Error(err))
	}

	h.logger.Debug("ConversationRecord: saved deferred LLM response",
		zap.String("trace_id", deferredResp.TraceID),
		zap.String("span_id", deferredResp.SpanID),
		zap.String("parent_span_id", parentSpanID))

	// 清除延迟响应信息
	ctx.WithValue(deferredResponseKey, nil)
}

// OnThinking 记录 LLM 思考过程
func (h *ConversationRecordHook) OnThinking(ctx *domain.HookContext, thinking string) {
	if thinking == "" {
		return
	}

	traceID := h.extractTraceID(ctx)
	if traceID == "" {
		traceID = trace.MustGetTraceID(ctx.Context)
	}

	// 获取父 span_id（优先使用 ToolParentSpanKey，其次 SpanKey）
	parentSpanID, _ := ctx.Get(ToolParentSpanKey).(string)
	if parentSpanID == "" {
		parentSpanID, _ = ctx.Get(SpanKey).(string)
	}

	spanID := h.idGenerator.Generate()

	// 记录思考过程，event_type 为 "thinking"
	record, err := h.createRecord(traceID, spanID, parentSpanID, "thinking", "assistant", thinking)
	if err != nil {
		h.logger.Error("Failed to create conversation record for thinking", zap.Error(err))
		return
	}

	// 设置范围信息
	if scope, ok := ctx.Get(ScopeKey).(ScopeInfo); ok {
		record.SetScope(scope.SessionKey, scope.UserCode, scope.AgentCode, scope.ChannelCode, scope.ChannelType)
	}

	if err := h.repo.Save(ctx.Context, record); err != nil {
		h.logger.Error("Failed to save conversation record for thinking", zap.Error(err))
		return
	}

	h.logger.Debug("ConversationRecord: saved thinking",
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
		zap.Int("thinking_len", len(thinking)))
}
