package claudecode

import (
	"context"
	"fmt"
	"sync"
	"time"

	claudecode "github.com/severity1/claude-agent-sdk-go"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/hook"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

func (p *ClaudeCodeProcessor) queryClaudeCodeStreaming(ctx context.Context, msg *bus.InboundMessage, userInput, cliSessionID, traceID string, provider *domain.LLMProvider, agent *domain.Agent, callback StreamingCallback) (string, *domain.Usage, error) {
	sessionKey := msg.SessionKey()

	// 创建工具钩子适配器
	var ccToolHookAdapter *toolHookAdapter
	var hookCtx *domain.HookContext
	var result string
	var llmCallCtx *domain.LLMCallContext
	var llmUsage = &domain.Usage{}

	if p.hookManager != nil {
		hookCtx = domain.NewHookContext(ctx)
		hookCtx.SetMetadata("session_key", sessionKey)
		hookCtx.SetMetadata("trace_id", traceID)
		// 启用思考过程，让 FeishuThinkingProcessHook 发送中间过程卡片
		hookCtx.SetMetadata("enable_thinking_process", "true")
		// 设置渠道信息，供 sendThinkingMessage 使用
		hookCtx.SetMetadata("chat_id", msg.ChatID)
		hookCtx.SetMetadata("channel_type", msg.Channel)

		userCode := ""
		agentCode := ""
		if agent != nil {
			agentCode = agent.AgentCode().String()
			userCode = agent.UserCode()
		}

		// 从 msg.Metadata 提取 channel_code
		channelCode := ""
		if msg.Metadata != nil {
			if v, ok := msg.Metadata["channel_code"].(string); ok {
				channelCode = v
			}
		}

		ccToolHookAdapter = &toolHookAdapter{
			bridge: &hook.ToolHookBridge{
				Manager:     p.hookManager,
				Logger:      p.logger,
				HookCtx:     hookCtx,
				SessionKey:  sessionKey,
				UserCode:    userCode,
				AgentCode:   agentCode,
				ChannelCode: channelCode,
				ChannelType: msg.Channel,
				TraceID:     traceID,
			},
		}

		// 构建 LLMCallContext 并调用 PreLLMCall hooks
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

		// 调用 PreLLMCall hooks
		modifiedCtx, err := p.hookManager.PreLLMCall(hookCtx, llmCallCtx)
		if err != nil {
			p.logger.Error("PreLLMCall failed", zap.Error(err))
		}
		if modifiedCtx != nil {
			llmCallCtx = modifiedCtx
		}

		// 确保 PostLLMCall 和 OnToolExecutionComplete 被调用
		// 使用 hookCtx（而非新建 context），确保 span 状态在 PreToolCall/PostToolCall 之间正确共享
		defer func() {
			resp := &domain.LLMResponse{Content: result, Usage: domain.Usage{}}
			if llmUsage != nil {
				resp.Usage = *llmUsage
			}
			p.hookManager.PostLLMCall(hookCtx, llmCallCtx, resp)
			// 工具执行完成后，写入延迟的最终 llm_response
			p.hookManager.OnToolExecutionComplete(hookCtx)
		}()
	}

	// 构建选项
	opts := p.buildOptions(provider, cliSessionID, agent, ccToolHookAdapter)

	p.logger.Info("开始 Claude Code 流式查询",
		zap.String("session_key", sessionKey),
		zap.String("cli_session_id", cliSessionID),
	)

	startTime := time.Now()

	// 使用 Client 接口进行流式处理
	client := claudecode.NewClient(opts...)

	if err := client.Connect(ctx); err != nil {
		p.logger.Error("Claude Code Connect 失败",
			zap.String("session_key", sessionKey),
			zap.Duration("duration", time.Since(startTime)),
			zap.Error(err),
		)
		return "", nil, fmt.Errorf("Claude Code Connect 失败: %w", err)
	}
	defer client.Disconnect()

	p.logger.Info("Claude Code 连接成功，开始流式查询",
		zap.String("session_key", sessionKey),
		zap.Duration("duration", time.Since(startTime)),
	)

	// 使用 QueryWithSession 发送查询
	sessionID := cliSessionID
	if sessionID == "" {
		sessionID = "default"
	}
	if err := client.QueryWithSession(ctx, userInput, sessionID); err != nil {
		p.logger.Error("Claude Code QueryWithSession 失败",
			zap.String("session_key", sessionKey),
			zap.Duration("duration", time.Since(startTime)),
			zap.Error(err),
		)
		return "", nil, fmt.Errorf("Claude Code QueryWithSession 失败: %w", err)
	}

	// 流式接收消息
	var cliSessionIDResult string
	var mu sync.Mutex

	msgChan := client.ReceiveMessages(ctx)
	for msg := range msgChan {
		if msg == nil {
			continue
		}

		switch m := msg.(type) {
		case *claudecode.AssistantMessage:
			for _, block := range m.Content {
				switch b := block.(type) {
				case *claudecode.TextBlock:
					mu.Lock()
					result += b.Text
					mu.Unlock()
					callback.OnText(b.Text)
				case *claudecode.ToolUseBlock:
					p.logger.Info("Claude Code 工具调用",
						zap.String("session_key", sessionKey),
						zap.String("tool_name", b.Name),
					)
					toolInput := make(map[string]any)
					if b.Input != nil {
						toolInput = b.Input
					}
					callback.OnToolCall(b.Name, toolInput)
				case *claudecode.ToolResultBlock:
					p.logger.Info("Claude Code 工具结果",
						zap.String("session_key", sessionKey),
						zap.Any("content", b.Content),
					)
					content := fmt.Sprintf("%v", b.Content)
					mu.Lock()
					result += content
					mu.Unlock()
					callback.OnToolResult("", content)
				case *claudecode.ThinkingBlock:
					// 只发送思考卡片，不累积到 result
					callback.OnThinking(b.Thinking)
				}
			}
		case *claudecode.ResultMessage:
			if m.SessionID != "" {
				cliSessionIDResult = m.SessionID
			}
			if m.IsError && m.Result != nil {
				result += fmt.Sprintf("\n[错误: %s]", *m.Result)
			}
			// 捕获 token usage（Claude CLI 在流式模式下 output_tokens 始终为 0）
			// 使用所有可用 token 字段求和作为兜底方案
			if m.Usage != nil && llmUsage != nil {
				llmUsage.PromptTokens = getUsageInt(*m.Usage, "input_tokens")
				llmUsage.CompletionTokens = getUsageInt(*m.Usage, "output_tokens")
				cacheRead := getUsageInt(*m.Usage, "cache_read_input_tokens")
				cacheCreate := getUsageInt(*m.Usage, "cache_creation_input_tokens")
				llmUsage.TotalTokens = llmUsage.PromptTokens + llmUsage.CompletionTokens + cacheRead + cacheCreate
				p.logger.Info("Token usage captured",
					zap.Any("usage", m.Usage),
					zap.Int("prompt", llmUsage.PromptTokens),
					zap.Int("completion", llmUsage.CompletionTokens),
					zap.Int("cache_read", cacheRead),
					zap.Int("cache_create", cacheCreate),
					zap.Int("total", llmUsage.TotalTokens),
				)
			} else {
				p.logger.Warn("Token usage not available",
					zap.Any("m.Usage", m.Usage),
					zap.Any("llmUsage", llmUsage),
				)
			}
			// ResultMessage 表示流式结束，立即调用 OnComplete 并退出
			callback.OnComplete(result)
			return cliSessionIDResult, llmUsage, nil
		case *claudecode.UserMessage:
			// 用户消息，不处理
		}
	}

	p.logger.Info("Claude Code 流式接收完成",
		zap.String("session_key", sessionKey),
		zap.Duration("duration", time.Since(startTime)),
	)

	// 如果循环正常结束（channel 关闭）但没有收到 ResultMessage，也调用 OnComplete
	callback.OnComplete(result)
	return cliSessionIDResult, llmUsage, nil
}

// queryClaudeCode 调用 Claude Code SDK
