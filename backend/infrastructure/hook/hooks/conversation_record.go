/**
 * ConversationRecordHook - 记录对话历史到 conversation_records 表
 * 记录 LLM 调用、工具调用等完整对话轨迹
 */
package hooks

import (
	"strings"
	"time"

	"github.com/weibh/taskmanager/domain"
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
	ScopeKey            contextKey = "conversation_scope"
	SpanKey             contextKey = "conversation_span"
	ToolParentSpanKey   contextKey = "conversation_tool_parent_span"
	promptKey           contextKey = "conversation_prompt"
	deferredResponseKey contextKey = "conversation_deferred_response"
)

// ScopeInfo 存储对话范围信息 - 导出供其他包使用
type ScopeInfo struct {
	SessionKey  string
	UserCode    string
	AgentCode   string
	ChannelCode string
	ChannelType string
}

// deferredLLMResponse 存储延迟的 LLM 响应信息
type deferredLLMResponse struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
	Content      string
	Usage        domain.Usage
	Scope        ScopeInfo
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
func (h *ConversationRecordHook) extractScope(ctx *domain.HookContext, callCtx *domain.LLMCallContext) ScopeInfo {
	scope := ScopeInfo{}

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

// containsToolCalls 检查 RawResponse 是否包含 tool_calls
func containsToolCalls(rawResponse string) bool {
	if rawResponse == "" {
		return false
	}
	return strings.Contains(rawResponse, `"tool_calls"`)
}
