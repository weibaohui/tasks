package claudecode

import (
	"context"
	"encoding/json"
	"fmt"

	claudecode "github.com/severity1/claude-agent-sdk-go"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/hook"
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

// toolHookAdapter bridges Claude Code SDK hooks to the domain hook system.
// It delegates the actual PreToolCall/PostToolCall logic to hook.ToolHookBridge
// so that both ClaudeCodeProcessor and OpenCodeProcessor share the same hook path.
type toolHookAdapter struct {
	bridge *hook.ToolHookBridge
}

// preToolUseAdapter converts Claude Code PreToolUse hook to domain.ToolHook
func (a *toolHookAdapter) preToolUseAdapter(ctx context.Context, input any, toolUseID *string, hookCtx claudecode.HookContext) (claudecode.HookJSONOutput, error) {
	preInput, ok := input.(*claudecode.PreToolUseHookInput)
	if !ok {
		return claudecode.HookJSONOutput{Continue: boolPtr(true)}, nil
	}

	if a.bridge != nil {
		a.bridge.PreToolCall(preInput.ToolName, preInput.ToolInput)
	}

	// Return continue=true to allow tool execution
	return claudecode.HookJSONOutput{Continue: boolPtr(true)}, nil
}

// postToolUseAdapter converts Claude Code PostToolUse hook to domain.ToolHook
func (a *toolHookAdapter) postToolUseAdapter(ctx context.Context, input any, toolUseID *string, hookCtx claudecode.HookContext) (claudecode.HookJSONOutput, error) {
	postInput, ok := input.(*claudecode.PostToolUseHookInput)
	if !ok {
		return claudecode.HookJSONOutput{Continue: boolPtr(true)}, nil
	}

	var output string
	if postInput.ToolResponse != nil {
		if respBytes, err := json.Marshal(postInput.ToolResponse); err == nil {
			output = string(respBytes)
		} else {
			output = fmt.Sprintf("%v", postInput.ToolResponse)
		}
	}

	if a.bridge != nil {
		a.bridge.PostToolCall(postInput.ToolName, postInput.ToolInput, output, true)
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
