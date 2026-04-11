package claudecode

import (
	"context"
	"encoding/json"
	"fmt"

	claudecode "github.com/severity1/claude-agent-sdk-go"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/hook"
	"github.com/weibh/taskmanager/infrastructure/hook/hooks"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

const defaultTokenAggregationLimit = 1000 // Token 聚合查询默认限制

// StreamingCallback 回调接口用于流式输出
type StreamingCallback interface {
	OnThinking(thinking string)
	OnToolCall(toolName string, input map[string]any)
	OnToolResult(toolName string, result string)
	OnText(text string)
	OnComplete(finalResult string)
	GetFinalResult() string
}

// toolHookAdapter bridges Claude Code SDK hooks to the domain hook system
type toolHookAdapter struct {
	hookManager *hook.Manager
	logger      *zap.Logger
	hookCtx     *domain.HookContext
	sessionKey  string
	userCode    string
	agentCode   string
	channelCode string
	channelType string
	traceID     string
}

// preToolUseAdapter converts Claude Code PreToolUse hook to domain.ToolHook
func (a *toolHookAdapter) preToolUseAdapter(ctx context.Context, input any, toolUseID *string, hookCtx claudecode.HookContext) (claudecode.HookJSONOutput, error) {
	preInput, ok := input.(*claudecode.PreToolUseHookInput)
	if !ok {
		a.logger.Warn("ClaudeCode PreToolUse: unexpected input type")
		return claudecode.HookJSONOutput{Continue: boolPtr(true)}, nil
	}

	// 在 Claude Code SDK 路径中，PostLLMCall（会设置 ToolParentSpanKey）在工具调用之后才执行（defer）。
	// 因此首次 PreToolCall 时 ToolParentSpanKey 还未设置，需要在这里主动设置，
	// 确保连续工具调用共享同一个父级（llm_response_with_tools），而不是互相嵌套。
	if a.hookCtx.Get(hooks.ToolParentSpanKey) == nil {
		if currentSpan, ok := a.hookCtx.Get(hooks.SpanKey).(string); ok && currentSpan != "" {
			a.hookCtx.WithValue(hooks.ToolParentSpanKey, currentSpan)
		}
	}

	// Convert to domain.ToolCallContext
	callCtx := &domain.ToolCallContext{
		ToolName:     preInput.ToolName,
		ToolInput:    preInput.ToolInput,
		SessionID:    a.sessionKey,
		TraceID:      a.traceID,
		SpanID:       "",
		ParentSpanID: "",
	}

	// Execute PreToolCall hooks
	_, err := a.hookManager.PreToolCall(a.hookCtx, callCtx)
	if err != nil {
		a.logger.Error("ClaudeCode PreToolUse hook failed",
			zap.String("tool", preInput.ToolName),
			zap.Error(err))
	}

	// Return continue=true to allow tool execution
	return claudecode.HookJSONOutput{Continue: boolPtr(true)}, nil
}

// postToolUseAdapter converts Claude Code PostToolUse hook to domain.ToolHook
func (a *toolHookAdapter) postToolUseAdapter(ctx context.Context, input any, toolUseID *string, hookCtx claudecode.HookContext) (claudecode.HookJSONOutput, error) {
	postInput, ok := input.(*claudecode.PostToolUseHookInput)
	if !ok {
		a.logger.Warn("ClaudeCode PostToolUse: unexpected input type")
		return claudecode.HookJSONOutput{Continue: boolPtr(true)}, nil
	}

	// Convert to domain.ToolCallContext
	callCtx := &domain.ToolCallContext{
		ToolName:     postInput.ToolName,
		ToolInput:    postInput.ToolInput,
		SessionID:    a.sessionKey,
		TraceID:      a.traceID,
		SpanID:       "",
		ParentSpanID: "",
	}

	// Convert tool response to ToolExecutionResult
	execResult := &domain.ToolExecutionResult{
		Success:  true,
		Duration: 0,
	}

	// Handle tool response
	if postInput.ToolResponse != nil {
		if respBytes, err := json.Marshal(postInput.ToolResponse); err == nil {
			execResult.Output = string(respBytes)
		} else {
			execResult.Output = fmt.Sprintf("%v", postInput.ToolResponse)
		}
	}

	// Execute PostToolCall hooks
	_, err := a.hookManager.PostToolCall(a.hookCtx, callCtx, execResult)
	if err != nil {
		a.logger.Error("ClaudeCode PostToolUse hook failed",
			zap.String("tool", postInput.ToolName),
			zap.Error(err))
	}

	// Return continue=true
	return claudecode.HookJSONOutput{Continue: boolPtr(true)}, nil
}

func boolPtr(b bool) *bool {
	return &b
}

// ClaudeCodeProcessor 处理 CodingAgent 类型消息的 Claude Code 会话
type ClaudeCodeProcessor struct {
	logger            *zap.Logger
	hookManager       *hook.Manager
	providerRepo      domain.LLMProviderRepository
	idGenerator       domain.IDGenerator
	requirementRepo   domain.RequirementRepository
	conversationRepo  domain.ConversationRecordRepository
	replicaCleanupSvc domain.ReplicaCleanupService
}

// ClaudeCodeProcessorInterface 定义 Claude Code 处理器的接口
type ClaudeCodeProcessorInterface interface {
	Process(ctx context.Context, msg *bus.InboundMessage, session *ClaudeCodeSession, agent *domain.Agent) (string, error)
	ProcessWithStreaming(ctx context.Context, msg *bus.InboundMessage, session *ClaudeCodeSession, agent *domain.Agent, callback StreamingCallback) error
}

// ClaudeCodeSession 会话上下文（包含 CLI Session ID）
type ClaudeCodeSession struct {
	SessionKey   string
	CliSessionID string
}

// NewClaudeCodeProcessor 创建 ClaudeCodeProcessor
func NewClaudeCodeProcessor(
	logger *zap.Logger,
	hookManager *hook.Manager,
	providerRepo domain.LLMProviderRepository,
	idGenerator domain.IDGenerator,
	requirementRepo domain.RequirementRepository,
	replicaCleanupSvc domain.ReplicaCleanupService,
	conversationRepo domain.ConversationRecordRepository,
) *ClaudeCodeProcessor {
	return &ClaudeCodeProcessor{
		logger:            logger,
		hookManager:       hookManager,
		providerRepo:      providerRepo,
		idGenerator:       idGenerator,
		requirementRepo:   requirementRepo,
		conversationRepo:  conversationRepo,
		replicaCleanupSvc: replicaCleanupSvc,
	}
}
