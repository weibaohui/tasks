package channel

import (
	"context"
	"fmt"
	"strings"

	claudecode "github.com/severity1/claude-agent-sdk-go"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/hook"
	"github.com/weibh/taskmanager/infrastructure/trace"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

// ClaudeCodeProcessor 处理 CodingAgent 类型消息的 Claude Code 会话
type ClaudeCodeProcessor struct {
	logger         *zap.Logger
	sessionManager *SessionManager
	hookManager    *hook.Manager
	providerRepo   domain.LLMProviderRepository
	idGenerator    domain.IDGenerator
	traceID        string
	spanID         string
}

// NewClaudeCodeProcessor 创建 ClaudeCodeProcessor
func NewClaudeCodeProcessor(
	logger *zap.Logger,
	sessionManager *SessionManager,
	hookManager *hook.Manager,
	providerRepo domain.LLMProviderRepository,
	idGenerator domain.IDGenerator,
) *ClaudeCodeProcessor {
	return &ClaudeCodeProcessor{
		logger:         logger,
		sessionManager: sessionManager,
		hookManager:    hookManager,
		providerRepo:   providerRepo,
		idGenerator:    idGenerator,
	}
}

// Process 处理 CodingAgent 消息
func (p *ClaudeCodeProcessor) Process(ctx context.Context, msg *bus.InboundMessage, agentCode, userCode string) (string, error) {
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

	// 保存用户消息到会话历史
	session := p.sessionManager.GetOrCreate(msg.SessionKey())
	session.AddMessage(Message{
		Role:    "user",
		Content: msg.Content,
		TraceID: traceID,
		SpanID:  spanID,
	})

	// 获取 Provider 配置（用于日志和配置）
	provider, err := p.providerRepo.FindDefaultActive(ctx, userCode)
	if err != nil {
		p.logger.Warn("获取 LLM Provider 失败，使用默认配置", zap.Error(err))
	}

	// 调用 Claude Code
	response, err := p.queryClaudeCode(ctx, msg.SessionKey(), msg.Content, provider)
	if err != nil {
		p.logger.Error("Claude Code 调用失败", zap.Error(err))
		return "", fmt.Errorf("Claude Code 调用失败: %w", err)
	}

	// 保存助手响应到会话历史
	session.AddMessage(Message{
		Role:    "assistant",
		Content: response,
		TraceID: traceID,
		SpanID:  spanID,
	})

	return response, nil
}

// queryClaudeCode 调用 Claude Code SDK
func (p *ClaudeCodeProcessor) queryClaudeCode(ctx context.Context, sessionKey, userInput string, provider *domain.LLMProvider) (string, error) {
	var result string
	var queryErr error

	// 构建 claudecode 选项
	opts := p.buildOptions(provider)

	p.logger.Info("开始 Claude Code 查询",
		zap.String("session_key", sessionKey),
		zap.String("provider", provider.ProviderKey()),
	)

	// 使用 QueryWithSession 创建或恢复会话
	err := claudecode.WithClient(ctx, func(client claudecode.Client) error {
		// 使用 sessionKey 作为 Context Session ID
		if err := client.QueryWithSession(ctx, userInput, sessionKey); err != nil {
			queryErr = err
			return err
		}

		// 流式接收响应
		result = p.streamResponse(ctx, client)
		return nil
	}, opts...)

	if err != nil {
		return "", err
	}
	if queryErr != nil {
		return "", queryErr
	}

	return result, nil
}

// buildOptions 根据 Provider 类型构建 claudecode 选项
func (p *ClaudeCodeProcessor) buildOptions(provider *domain.LLMProvider) []claudecode.Option {
	opts := []claudecode.Option{}

	p.logger.Info("Claude Code 选项配置",
		zap.String("provider", provider.ProviderKey()),
		zap.String("model", provider.DefaultModel()),
		zap.Int("options_count", len(opts)),
	)
	if provider != nil && provider.ProviderType() == domain.ProviderTypeAnthropic {

		// 设置 API Key 环境变量
		if provider.APIKey() != "" {
			opts = append(opts, claudecode.WithEnv(map[string]string{
				"ANTHROPIC_API_KEY":  provider.APIKey(),
				"ANTHROPIC_BASE_URL": provider.APIBase(),
			}))
		}

		// 设置模型
		if provider.DefaultModel() != "" {
			opts = append(opts, claudecode.WithModel(provider.DefaultModel()))
		}

	}

	return opts
}

// streamResponse 流式接收 Claude Code 响应
func (p *ClaudeCodeProcessor) streamResponse(ctx context.Context, client claudecode.Client) string {
	var response strings.Builder
	msgChan := client.ReceiveMessages(ctx)

	for {
		select {
		case message := <-msgChan:
			if message == nil {
				return response.String()
			}

			switch msg := message.(type) {
			case *claudecode.AssistantMessage:
				for _, block := range msg.Content {
					if textBlock, ok := block.(*claudecode.TextBlock); ok {
						response.WriteString(textBlock.Text)
					}
				}
			case *claudecode.ResultMessage:
				if msg.IsError {
					p.logger.Error("Claude Code 返回错误",
						zap.String("trace_id", p.traceID),
						zap.String("span_id", p.spanID),
					)
					if msg.Result != nil {
						response.WriteString(fmt.Sprintf("\n[错误: %s]", *msg.Result))
					}
				}
				p.logger.Info("Claude Code 查询完成",
					zap.String("trace_id", p.traceID),
					zap.String("span_id", p.spanID),
				)
				return response.String()
			}
		case <-ctx.Done():
			p.logger.Info("Claude Code 查询上下文取消",
				zap.String("trace_id", p.traceID),
				zap.String("span_id", p.spanID),
			)
			return response.String()
		}
	}
}
