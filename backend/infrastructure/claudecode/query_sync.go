package claudecode

import (
	"context"
	"fmt"
	"time"

	claudecode "github.com/severity1/claude-agent-sdk-go"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

func (p *ClaudeCodeProcessor) queryClaudeCode(ctx context.Context, msg *bus.InboundMessage, cliSessionID, traceID string, provider *domain.LLMProvider, agent *domain.Agent) (string, string, error) {
	sessionKey := msg.SessionKey()
	userInput := msg.Content

	// 创建工具钩子适配器，用于将 Claude Code SDK 工具调用桥接到现有 hook 系统
	var ccToolHookAdapter *toolHookAdapter
	if p.hookManager != nil {
		hookCtx := domain.NewHookContext(ctx)
		hookCtx.SetMetadata("session_key", sessionKey)
		hookCtx.SetMetadata("trace_id", traceID)

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
			hookManager: p.hookManager,
			logger:      p.logger,
			hookCtx:     hookCtx,
			sessionKey:  sessionKey,
			userCode:    userCode,
			agentCode:   agentCode,
			channelCode: channelCode,
			channelType: msg.Channel,
			traceID:     traceID,
		}
	}

	// 构建 claudecode 选项
	opts := p.buildOptions(provider, cliSessionID, agent, ccToolHookAdapter)

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

	// 使用 Client 接口进行流式处理
	client := claudecode.NewClient(opts...)

	// 使用 Connect 建立连接
	if err := client.Connect(ctx); err != nil {
		p.logger.Error("Claude Code Connect 失败",
			zap.String("session_key", sessionKey),
			zap.Duration("duration", time.Since(startTime)),
			zap.Error(err),
		)
		return "", "", fmt.Errorf("Claude Code Connect 失败: %w", err)
	}
	defer client.Disconnect()

	p.logger.Info("Claude Code 连接成功，开始查询",
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
		return "", "", fmt.Errorf("Claude Code QueryWithSession 失败: %w", err)
	}

	// 使用 ReceiveMessages 接收流式消息
	var result string
	var cliSessionIDResult string

	msgChan := client.ReceiveMessages(ctx)
	resultChan := make(chan string, 1)
	sessionChan := make(chan string, 1)

	// 启动 goroutine 处理消息
	go func() {
		var result string
		var cliSessionIDResult string
		for msg := range msgChan {
			if msg == nil {
				continue
			}

			switch m := msg.(type) {
			case *claudecode.AssistantMessage:
				for _, block := range m.Content {
					switch b := block.(type) {
					case *claudecode.TextBlock:
						result += b.Text
					case *claudecode.ToolUseBlock:
						// 记录工具调用
						p.logger.Info("Claude Code 工具调用",
							zap.String("session_key", sessionKey),
							zap.String("tool_name", b.Name),
						)
						result += fmt.Sprintf("\n[调用工具: %s]\n", b.Name)
					case *claudecode.ToolResultBlock:
						// 记录工具结果
						p.logger.Info("Claude Code 工具结果",
							zap.String("session_key", sessionKey),
							zap.Any("content", b.Content),
						)
						result += fmt.Sprintf("%v\n", b.Content)
					case *claudecode.ThinkingBlock:
						// 思考过程
						result += fmt.Sprintf("\n[思考: %s]\n", b.Thinking)
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
				// ResultMessage 表示会话结束，跳出循环
				resultChan <- result
				sessionChan <- cliSessionIDResult
				return
			case *claudecode.UserMessage:
				// 用户消息，不处理
			}
		}
		// 如果通道正常关闭但没有 ResultMessage，发送空结果
		resultChan <- result
		sessionChan <- cliSessionIDResult
	}()

	// 等待结果或超时
	select {
	case result = <-resultChan:
		cliSessionIDResult = <-sessionChan
	case <-ctx.Done():
		return "", "", fmt.Errorf("Claude Code 查询超时: %w", ctx.Err())
	}

	p.logger.Info("Claude Code 流式接收完成",
		zap.String("session_key", sessionKey),
		zap.Duration("duration", time.Since(startTime)),
	)

	return result, cliSessionIDResult, nil
}

// buildOptions 根据 Provider 类型构建 claudecode 选项
