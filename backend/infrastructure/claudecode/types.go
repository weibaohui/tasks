package claudecode

import (
	"context"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/hook"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

// StreamingCallback 回调接口用于流式输出
type StreamingCallback interface {
	OnStart()
	OnThinking(thinking string)
	OnToolCall(toolName string, input map[string]any)
	OnToolResult(toolName string, result string)
	OnText(text string)
	OnComplete(finalResult string)
	OnError(err error)
	GetFinalResult() string
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
