/**
 * ConversationRecordHook - 记录对话历史到 conversation_records 表
 * 记录 LLM 调用、工具调用等完整对话轨迹
 */
package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/trace"
	"go.uber.org/zap"
)

// ConversationRecordHook 配置
type ConversationRecordHookConfig struct {
	// SessionKeyExtractor 从 HookContext 提取 session_key
	SessionKeyExtractor func(ctx *domain.HookContext) string
	// UserCodeExtractor 从 HookContext 提取 user_code
	UserCodeExtractor func(ctx *domain.HookContext) string
	// AgentCodeExtractor 从 HookContext 提取 agent_code
	AgentCodeExtractor func(ctx *domain.HookContext) string
	// ChannelCodeExtractor 从 HookContext 提取 channel_code
	ChannelCodeExtractor func(ctx *domain.HookContext) string
	// ChannelTypeExtractor 从 HookContext 提取 channel_type
	ChannelTypeExtractor func(ctx *domain.HookContext) string
}

// ConversationRecordHook 记录对话历史的 Hook
type ConversationRecordHook struct {
	*domain.BaseHook
	repo        domain.ConversationRecordRepository
	idGenerator domain.IDGenerator
	logger      *zap.Logger
	config      *ConversationRecordHookConfig
}

// NewConversationRecordHook 创建 ConversationRecordHook
func NewConversationRecordHook(
	repo domain.ConversationRecordRepository,
	idGenerator domain.IDGenerator,
	logger *zap.Logger,
	config *ConversationRecordHookConfig,
) *ConversationRecordHook {
	if config == nil {
		config = &ConversationRecordHookConfig{}
	}

	return &ConversationRecordHook{
		BaseHook:    domain.NewBaseHook("conversation_record", 50, domain.HookTypeLLM),
		repo:        repo,
		idGenerator: idGenerator,
		logger:      logger,
		config:      config,
	}
}

// contextKey 用于在 HookContext 中存储和获取数据
type contextKey string

const (
	scopeKey            contextKey = "conversation_scope"
	spanKey             contextKey = "conversation_span"
	promptKey           contextKey = "conversation_prompt"
	deferredResponseKey contextKey = "conversation_deferred_response"
)

// scopeInfo 存储对话范围信息
type scopeInfo struct {
	SessionKey  string
	UserCode    string
	AgentCode   string
	ChannelCode string
	ChannelType string
}

// deferredLLMResponse 存储延迟的 LLM 响应信息
type deferredLLMResponse struct {
	TraceID    string
	SpanID     string
	ParentSpanID string
	Content    string
	Usage      domain.Usage
	Scope      scopeInfo
}

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

	// 提取范围信息
	scope := h.extractScope(ctx, callCtx)
	ctx.WithValue(scopeKey, scope)
	ctx.WithValue(spanKey, spanID)
	ctx.WithValue(promptKey, callCtx.Prompt)

	// 把 scope 存回 callCtx.Metadata，这样 PostLLMCall 能通过 extractScope 拿到
	if callCtx.Metadata == nil {
		callCtx.Metadata = make(map[string]string)
	}
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
	if err := h.repo.Save(context.Background(), record); err != nil {
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

	// 获取用户输入的 span_id 作为 parent（来自 PreLLMCall 存储的 spanKey）
	parentSpanID, _ := ctx.Get(spanKey).(string)
	if parentSpanID == "" {
		parentSpanID = trace.GetSpanID(ctx.Context)
	}

	// 检查 resp.RawResponse 是否包含 tool_calls（LLM 决定调用工具的标志）
	hasToolCalls := containsToolCalls(resp.RawResponse)

	// LLM 响应生成新的 span_id，parent 指向上游用户输入
	spanID := h.idGenerator.Generate()

	// 存储新的 span_id 到 HookContext.values，供后续 PreToolCall 使用
	ctx.WithValue(spanKey, spanID)

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

	// 如果 LLM 决定调用工具，先记录中间响应（包含 tool_calls）
	if hasToolCalls && resp.Content != "" {
		toolCallsSpanID := h.idGenerator.Generate()
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

		// 更新 span 链：后续的 tool_call 应该以这个中间响应为 parent
		ctx.WithValue(spanKey, toolCallsSpanID)

		// 延迟记录最终的 llm_response：存储信息到 context，由 OnToolExecutionComplete 记录
		ctx.WithValue(deferredResponseKey, &deferredLLMResponse{
			TraceID:      traceID,
			SpanID:       spanID,
			ParentSpanID: "", // 将在 OnToolExecutionComplete 时设置为 tool_call 的 span
			Content:      resp.Content,
			Usage:        resp.Usage,
			Scope: scopeInfo{
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

	if err := h.repo.Save(context.Background(), record); err != nil {
		h.logger.Error("Failed to save conversation record for LLM response", zap.Error(err))
	}

	h.logger.Debug("ConversationRecord: saved LLM response",
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
		zap.String("role", role),
		zap.Int("content_len", len(content)))

	return resp, nil
}

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

	// 从 ctx 获取父 span_id（PostLLMCall 会存储新的 span）
	parentSpanID := ""
	if p, ok := ctx.Get(spanKey).(string); ok {
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
	if scope, ok := ctx.Get(scopeKey).(scopeInfo); ok {
		record.SetScope(scope.SessionKey, scope.UserCode, scope.AgentCode, scope.ChannelCode, scope.ChannelType)
	}

	if err := h.repo.Save(context.Background(), record); err != nil {
		h.logger.Error("Failed to save conversation record for tool call", zap.Error(err))
	}

	// 存储当前 span_id 供 PostToolCall 使用
	ctx.WithValue(spanKey, spanID)

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
	toolCallSpanID, _ := ctx.Get(spanKey).(string)
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

	// 设置范围 - 从 scopeKey 获取
	if scope, ok := ctx.Get(scopeKey).(scopeInfo); ok {
		record.SetScope(scope.SessionKey, scope.UserCode, scope.AgentCode, scope.ChannelCode, scope.ChannelType)
	}

	if err := h.repo.Save(context.Background(), record); err != nil {
		h.logger.Error("Failed to save conversation record for tool result", zap.Error(err))
	}

	h.logger.Debug("ConversationRecord: saved tool result",
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
		zap.String("parent_span_id", toolCallSpanID),
		zap.String("tool_name", callCtx.ToolName),
		zap.Bool("success", result.Success))

	// 更新 ctx 中的 span_id 为新的 tool_result span_id，供后续调用使用
	ctx.WithValue(spanKey, spanID)

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
	toolCallSpanID, _ := ctx.Get(spanKey).(string)
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
	if scope, ok := ctx.Get(scopeKey).(scopeInfo); ok {
		record.SetScope(scope.SessionKey, scope.UserCode, scope.AgentCode, scope.ChannelCode, scope.ChannelType)
	}

	if err := h.repo.Save(context.Background(), record); err != nil {
		h.logger.Error("Failed to save conversation record for tool error", zap.Error(err))
	}

	h.logger.Debug("ConversationRecord: saved tool error",
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
		zap.String("parent_span_id", toolCallSpanID),
		zap.String("tool_name", callCtx.ToolName),
		zap.Error(err))

	// 更新 ctx 中的 span_id 为新的 tool_error span_id，供后续调用使用
	ctx.WithValue(spanKey, spanID)

	return &domain.ToolExecutionResult{Success: false, Error: err}, nil
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
// 此时应该记录最终的 llm_response，parent 应为 tool_call 的 span
func (h *ConversationRecordHook) OnToolExecutionComplete(ctx *domain.HookContext) {
	// 获取延迟的 LLM 响应信息
	deferredResp, ok := ctx.Get(deferredResponseKey).(*deferredLLMResponse)
	if !ok || deferredResp == nil {
		return
	}

	// 获取 tool_call 的 span_id 作为 parent
	// PostToolCall 会将 tool_call 的 span 存储到 spanKey
	parentSpanID, _ := ctx.Get(spanKey).(string)
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

	if err := h.repo.Save(context.Background(), record); err != nil {
		h.logger.Error("Failed to save conversation record for deferred LLM response", zap.Error(err))
	}

	h.logger.Debug("ConversationRecord: saved deferred LLM response",
		zap.String("trace_id", deferredResp.TraceID),
		zap.String("span_id", deferredResp.SpanID),
		zap.String("parent_span_id", parentSpanID))

	// 清除延迟响应信息
	ctx.WithValue(deferredResponseKey, nil)
}

// createRecord 创建 conversation record
func (h *ConversationRecordHook) createRecord(traceID, spanID, parentSpanID, eventType, role, content string) (*domain.ConversationRecord, error) {
	id := domain.NewConversationRecordID(h.idGenerator.Generate())

	record, err := domain.NewConversationRecord(id, traceID, eventType)
	if err != nil {
		return nil, err
	}

	record.SetSpan(spanID, parentSpanID)
	record.SetMessage(role, content)
	record.SetTimestamp(time.Now())

	return record, nil
}

// extractScope 从 context 和 callCtx 提取范围信息
func (h *ConversationRecordHook) extractScope(ctx *domain.HookContext, callCtx *domain.LLMCallContext) scopeInfo {
	scope := scopeInfo{}

	// 优先从 callCtx.Metadata 提取（由 processor.go 直接设置）
	if callCtx != nil && callCtx.Metadata != nil {
		scope.SessionKey = callCtx.Metadata["session_key"]
		scope.UserCode = callCtx.Metadata["user_code"]
		scope.AgentCode = callCtx.Metadata["agent_code"]
		scope.ChannelCode = callCtx.Metadata["channel_code"]
		scope.ChannelType = callCtx.Metadata["channel_type"]
	}

	// 如果 Metadata 没有，则尝试从 ctx extractors 提取
	if scope.SessionKey == "" && h.config.SessionKeyExtractor != nil && ctx != nil {
		scope.SessionKey = h.config.SessionKeyExtractor(ctx)
	} else if scope.SessionKey == "" && callCtx != nil {
		scope.SessionKey = callCtx.SessionID
	}

	if scope.UserCode == "" && h.config.UserCodeExtractor != nil && ctx != nil {
		scope.UserCode = h.config.UserCodeExtractor(ctx)
	}

	if scope.AgentCode == "" && h.config.AgentCodeExtractor != nil && ctx != nil {
		scope.AgentCode = h.config.AgentCodeExtractor(ctx)
	}

	if scope.ChannelCode == "" && h.config.ChannelCodeExtractor != nil && ctx != nil {
		scope.ChannelCode = h.config.ChannelCodeExtractor(ctx)
	}

	if scope.ChannelType == "" && h.config.ChannelTypeExtractor != nil && ctx != nil {
		scope.ChannelType = h.config.ChannelTypeExtractor(ctx)
	}

	return scope
}

// extractTraceID 从 context 提取 trace_id
func (h *ConversationRecordHook) extractTraceID(ctx *domain.HookContext) string {
	if ctx == nil {
		return ""
	}
	return ctx.GetMetadata("trace_id")
}

// join 拼接字符串
func join(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// containsToolCalls 检查 RawResponse 是否包含 tool_calls
func containsToolCalls(rawResponse string) bool {
	if rawResponse == "" {
		return false
	}
	return strings.Contains(rawResponse, `"tool_calls"`)
}
