package claudecode

import (
	"context"
	"fmt"
	"time"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/claudecode/cli"
	"github.com/weibh/taskmanager/infrastructure/hook"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

func (p *ClaudeCodeProcessor) queryClaudeCode(ctx context.Context, msg *bus.InboundMessage, cliSessionID, traceID string, provider *domain.LLMProvider, agent *domain.Agent) (string, string, error) {
	sessionKey := msg.SessionKey()
	userInput := msg.Content

	// 创建 CLI 处理器
	cliProcessor := cli.NewProcessor(p.logger)

	// 获取配置
	var config *domain.ClaudeCodeConfig
	if agent != nil {
		config = agent.ClaudeCodeConfig()
	}
	if config == nil {
		config = domain.DefaultClaudeCodeConfig()
	}

	// 创建工具钩子桥接器
	var hookCtx *domain.HookContext
	var toolAdapter *hook.ToolHookBridge

	if p.hookManager != nil {
		hookCtx = domain.NewHookContext(ctx)
		hookCtx.SetMetadata("session_key", sessionKey)
		hookCtx.SetMetadata("trace_id", traceID)

		userCode := ""
		agentCode := ""
		if agent != nil {
			agentCode = agent.AgentCode().String()
			userCode = agent.UserCode()
		}

		channelCode := ""
		if msg.Metadata != nil {
			if v, ok := msg.Metadata["channel_code"].(string); ok {
				channelCode = v
			}
		}

		toolAdapter = &hook.ToolHookBridge{
			Manager:     p.hookManager,
			Logger:      p.logger,
			HookCtx:     hookCtx,
			SessionKey:  sessionKey,
			UserCode:    userCode,
			AgentCode:   agentCode,
			ChannelCode: channelCode,
			ChannelType: msg.Channel,
			TraceID:     traceID,
		}
	}

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

	// 包装 callback 以桥接 tool hooks
	var callback *syncCallback
	if toolAdapter != nil {
		callback = &syncCallback{
			toolAdapter: toolAdapter,
		}
	} else {
		callback = &syncCallback{}
	}

	// 执行查询
	result, tokenUsage, newSessionID, err := cliProcessor.QueryStreaming(
		ctx, msg, userInput, cliSessionID, traceID, provider, config, callback,
	)

	if err != nil {
		p.logger.Error("Claude Code 查询失败",
			zap.String("session_key", sessionKey),
			zap.Duration("duration", time.Since(startTime)),
			zap.Error(err),
		)
		return "", "", fmt.Errorf("Claude Code 查询失败: %w", err)
	}

	p.logger.Info("Claude Code 查询完成",
		zap.String("session_key", sessionKey),
		zap.Duration("duration", time.Since(startTime)),
		zap.String("result_length", fmt.Sprintf("%d", len(result))),
	)

	if tokenUsage != nil {
		p.logger.Info("Token usage",
			zap.Int("prompt", tokenUsage.InputTokens),
			zap.Int("completion", tokenUsage.OutputTokens),
			zap.Int("total", tokenUsage.Total),
		)
	}

	return result, newSessionID, nil
}

// syncCallback 同步回调，用于将流式结果同步化
type syncCallback struct {
	toolAdapter *hook.ToolHookBridge
	result      string
}

func (c *syncCallback) OnStart() {}
func (c *syncCallback) OnThinking(thinking string) {}
func (c *syncCallback) OnToolCall(toolName string, input map[string]any) {
	if c.toolAdapter != nil {
		c.toolAdapter.PreToolCall(toolName, input)
	}
}
func (c *syncCallback) OnToolResult(toolName string, result string) {
	if c.toolAdapter != nil {
		c.toolAdapter.PostToolCall(toolName, nil, result, true)
	}
}
func (c *syncCallback) OnText(text string) {
	c.result += text
}
func (c *syncCallback) OnComplete(finalResult string) {
	c.result = finalResult
}
func (c *syncCallback) OnError(err error) {}
func (c *syncCallback) GetFinalResult() string { return c.result }
