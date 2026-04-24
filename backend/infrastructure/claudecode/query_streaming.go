package claudecode

import (
	"context"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/claudecode/cli"
	"github.com/weibh/taskmanager/infrastructure/hook"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

func (p *ClaudeCodeProcessor) queryClaudeCodeStreaming(ctx context.Context, msg *bus.InboundMessage, userInput, cliSessionID, traceID string, provider *domain.LLMProvider, agent *domain.Agent, callback StreamingCallback) (string, *domain.Usage, error) {
	sessionKey := msg.SessionKey()

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
	var llmCallCtx *domain.LLMCallContext
	var toolAdapter *hook.ToolHookBridge
	var result string
	var llmUsage = &domain.Usage{}

	if p.hookManager != nil {
		hookCtx = domain.NewHookContext(ctx)
		hookCtx.SetMetadata("session_key", sessionKey)
		hookCtx.SetMetadata("trace_id", traceID)
		hookCtx.SetMetadata("enable_thinking_process", "true")
		hookCtx.SetMetadata("chat_id", msg.ChatID)
		hookCtx.SetMetadata("channel_type", msg.Channel)

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

		llmCallCtx = &domain.LLMCallContext{
			Prompt:    userInput,
			UserInput: userInput,
			Model:     "claude_code",
			SessionID: sessionKey,
			TraceID:   traceID,
			Metadata: map[string]string{
				"session_key":  sessionKey,
				"trace_id":     traceID,
				"user_code":    userCode,
				"agent_code":   agentCode,
				"channel_code": channelCode,
				"channel_type": msg.Channel,
				"chat_id":      msg.ChatID,
			},
		}

		modifiedCtx, err := p.hookManager.PreLLMCall(hookCtx, llmCallCtx)
		if err != nil {
			p.logger.Error("PreLLMCall failed", zap.Error(err))
		}
		if modifiedCtx != nil {
			llmCallCtx = modifiedCtx
		}

		// 包装 callback 以桥接 tool hooks
		wrappedCallback := &toolHookCallback{
			StreamingCallback: callback,
			toolAdapter:        toolAdapter,
		}

		defer func() {
			resp := &domain.LLMResponse{Content: result, Usage: domain.Usage{}}
			if llmUsage != nil {
				resp.Usage = *llmUsage
			}
			p.hookManager.PostLLMCall(hookCtx, llmCallCtx, resp)
			p.hookManager.OnToolExecutionComplete(hookCtx)
		}()

		result, tokenUsage, _, err := cliProcessor.QueryStreaming(
			ctx, msg, userInput, cliSessionID, traceID, provider, config, wrappedCallback,
		)
		if err != nil {
			return "", nil, err
		}
		if tokenUsage != nil {
			llmUsage.PromptTokens = tokenUsage.InputTokens
			llmUsage.CompletionTokens = tokenUsage.OutputTokens
			llmUsage.TotalTokens = tokenUsage.Total
		}
		return result, llmUsage, nil
	}

	result, tokenUsage, _, err := cliProcessor.QueryStreaming(
		ctx, msg, userInput, cliSessionID, traceID, provider, config, callback,
	)
	if err != nil {
		return "", nil, err
	}
	if tokenUsage != nil {
		llmUsage.PromptTokens = tokenUsage.InputTokens
		llmUsage.CompletionTokens = tokenUsage.OutputTokens
		llmUsage.TotalTokens = tokenUsage.Total
	}
	return result, llmUsage, nil
}

// toolHookCallback 包装 StreamingCallback，桥接 tool hooks
type toolHookCallback struct {
	StreamingCallback
	toolAdapter *hook.ToolHookBridge
}

func (c *toolHookCallback) OnToolCall(toolName string, input map[string]any) {
	if c.toolAdapter != nil {
		c.toolAdapter.PreToolCall(toolName, input)
	}
	c.StreamingCallback.OnToolCall(toolName, input)
}

func (c *toolHookCallback) OnToolResult(toolName string, result string) {
	c.StreamingCallback.OnToolResult(toolName, result)
	if c.toolAdapter != nil {
		c.toolAdapter.PostToolCall(toolName, nil, result, true)
	}
}

func (c *toolHookCallback) OnComplete(finalResult string) {
	c.StreamingCallback.OnComplete(finalResult)
}

// queryClaudeCode 调用 Claude Code SDK
