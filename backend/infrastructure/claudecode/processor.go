package claudecode

import (
	"context"
	"errors"
	"fmt"
	"time"

	claudecode "github.com/severity1/claude-agent-sdk-go"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/hook"
	"github.com/weibh/taskmanager/infrastructure/trace"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

// ClaudeCodeProcessor 处理 CodingAgent 类型消息的 Claude Code 会话
type ClaudeCodeProcessor struct {
	logger       *zap.Logger
	hookManager  *hook.Manager
	providerRepo domain.LLMProviderRepository
	idGenerator  domain.IDGenerator
	traceID      string
	spanID       string
}

// ClaudeCodeProcessorInterface 定义 Claude Code 处理器的接口
type ClaudeCodeProcessorInterface interface {
	Process(ctx context.Context, msg *bus.InboundMessage, session *ClaudeCodeSession, agentCode, userCode string) (string, error)
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
) *ClaudeCodeProcessor {
	return &ClaudeCodeProcessor{
		logger:       logger,
		hookManager:  hookManager,
		providerRepo: providerRepo,
		idGenerator:  idGenerator,
	}
}

// Process 处理 CodingAgent 消息
func (p *ClaudeCodeProcessor) Process(ctx context.Context, msg *bus.InboundMessage, session *ClaudeCodeSession, agentCode, userCode string) (string, error) {
	// 生成 trace 信息
	ctx, traceID, spanID := trace.StartTrace(ctx)
	p.traceID = traceID
	p.spanID = spanID

	preview := msg.Content
	if len(preview) > 80 {
		preview = preview[:80] + "..."
	}
	p.logger.Info("ClaudeCode 处理消息",
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
		zap.String("session_key", msg.SessionKey()),
		zap.String("agent_code", agentCode),
		zap.String("内容", preview),
	)

	// 获取 Provider 配置（用于日志和配置）
	provider, err := p.providerRepo.FindDefaultActive(ctx, userCode)
	if err != nil {
		p.logger.Warn("获取 LLM Provider 失败，使用默认配置", zap.Error(err))
		provider = nil
	}

	// 调用 Claude Code（使用独立的 context，避免被取消）
	queryCtx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// 从会话获取 CLI Session UUID，用于会话恢复
	cliSessionID := ""
	if session != nil {
		cliSessionID = session.CliSessionID
	}

	// 调用 Claude Code
	response, newCliSessionID, err := p.queryClaudeCode(queryCtx, msg.SessionKey(), msg.Content, cliSessionID, provider)
	if err != nil {
		p.logger.Error("Claude Code 调用失败", zap.Error(err))
		return "", fmt.Errorf("Claude Code 调用失败: %w", err)
	}

	// 如果返回了新的 CLI Session ID，更新会话
	if newCliSessionID != "" && session != nil {
		session.CliSessionID = newCliSessionID
		p.logger.Info("Claude Code 会话已保存",
			zap.String("session_key", msg.SessionKey()),
			zap.String("cli_session_id", newCliSessionID),
		)
	}

	p.logger.Info("Claude Code 返回响应",
		zap.String("session_key", msg.SessionKey()),
		zap.String("response_length", fmt.Sprintf("%d", len(response))),
		zap.String("response_preview", func() string {
			if len(response) > 100 {
				return response[:100] + "..."
			}
			return response
		}()),
	)

	return response, nil
}

// queryClaudeCode 调用 Claude Code SDK
func (p *ClaudeCodeProcessor) queryClaudeCode(ctx context.Context, sessionKey, userInput, cliSessionID string, provider *domain.LLMProvider) (string, string, error) {
	// 构建 claudecode 选项
	opts := p.buildOptions(provider, cliSessionID)

	p.logger.Info("开始 Claude Code 查询",
		zap.String("session_key", sessionKey),
		zap.String("cli_session_id", cliSessionID),
		zap.String("provider", func() string {
			if provider != nil {
				return provider.ProviderKey()
			}
			return "default"
		}()),
	)

	startTime := time.Now()

	// 使用 Query 创建查询 iterator（类似 quickstart 例子）
	iterator, err := claudecode.Query(ctx, userInput, opts...)
	if err != nil {
		p.logger.Error("Claude Code Query 失败",
			zap.String("session_key", sessionKey),
			zap.Duration("duration", time.Since(startTime)),
			zap.Error(err),
		)
		return "", "", fmt.Errorf("Claude Code Query 失败: %w", err)
	}
	defer iterator.Close()

	p.logger.Info("Claude Code Query 执行成功，开始接收消息",
		zap.String("session_key", sessionKey),
		zap.Duration("duration", time.Since(startTime)),
	)

	// 使用 iterator 接收消息
	var result string
	var cliSessionIDResult string

	for {
		msg, err := iterator.Next(ctx)
		if err != nil {
			if errors.Is(err, claudecode.ErrNoMoreMessages) {
				p.logger.Info("Claude Code 消息接收完成",
					zap.String("session_key", sessionKey),
					zap.Duration("duration", time.Since(startTime)),
				)
				break
			}
			p.logger.Error("Claude Code 接收消息失败",
				zap.String("session_key", sessionKey),
				zap.Error(err),
			)
			return "", "", fmt.Errorf("接收消息失败: %w", err)
		}

		if msg == nil {
			p.logger.Info("Claude Code 收到 nil 消息，结束",
				zap.String("session_key", sessionKey),
			)
			break
		}

		switch m := msg.(type) {
		case *claudecode.AssistantMessage:
			for _, block := range m.Content {
				if textBlock, ok := block.(*claudecode.TextBlock); ok {
					result += textBlock.Text
				}
			}
		case *claudecode.ResultMessage:
			if m.SessionID != "" {
				cliSessionIDResult = m.SessionID
			}
			if m.IsError && m.Result != nil {
				result += fmt.Sprintf("\n[错误: %s]", *m.Result)
			}
			p.logger.Info("Claude Code ResultMessage",
				zap.String("session_key", sessionKey),
				zap.String("cli_session_id", cliSessionIDResult),
				zap.Bool("is_error", m.IsError),
			)
		}
	}

	return result, cliSessionIDResult, nil
}

// buildOptions 根据 Provider 类型构建 claudecode 选项
func (p *ClaudeCodeProcessor) buildOptions(provider *domain.LLMProvider, cliSessionID string) []claudecode.Option {
	opts := []claudecode.Option{}

	// 使用 MiniMax API（与 quickstart 示例相同）
	opts = append(opts, claudecode.WithEnv(map[string]string{
		"ANTHROPIC_API_KEY":  "sk-4e7ehiWvl3EDNcwsZ4ul8mR9AjLGH1DNBqInGlBHyD2ZkIwB",
		"ANTHROPIC_BASE_URL": "https://minimax.a7m.com.cn",
	}))
	opts = append(opts, claudecode.WithModel("MiniMax-M2.7-highspeed"))

	// 如果有 CLI Session ID，使用 WithResume 恢复会话
	if cliSessionID != "" {
		opts = append(opts, claudecode.WithResume(cliSessionID))
	}

	p.logger.Info("Claude Code 选项配置",
		zap.String("provider", func() string {
			if provider != nil {
				return provider.ProviderKey()
			}
			return "default"
		}()),
		zap.String("cli_session_id", cliSessionID),
		zap.Int("options_count", len(opts)),
	)

	return opts
}
