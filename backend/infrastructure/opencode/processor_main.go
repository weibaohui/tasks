package opencode

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/hook"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

// OpenCodeSession 会话上下文
type OpenCodeSession struct {
	SessionKey   string
	CliSessionID string
	WorkDir      string
}

// OpenCodeProcessor 处理 OpenCodeAgent 类型消息的处理器
type OpenCodeProcessor struct {
	logger            *zap.Logger
	hookManager       *hook.Manager
	providerRepo      domain.LLMProviderRepository
	requirementRepo   domain.RequirementRepository
	conversationRepo  domain.ConversationRecordRepository
	replicaCleanupSvc domain.ReplicaCleanupService
}

// OpenCodeProcessorInterface 定义 OpenCode 处理器的接口
type OpenCodeProcessorInterface interface {
	Process(ctx context.Context, msg *bus.InboundMessage, session *OpenCodeSession, agent *domain.Agent) (string, error)
	ProcessWithStreaming(ctx context.Context, msg *bus.InboundMessage, session *OpenCodeSession, agent *domain.Agent, callback StreamingCallback) error
}

// NewOpenCodeProcessor 创建 OpenCodeProcessor
func NewOpenCodeProcessor(
	logger *zap.Logger,
	hookManager *hook.Manager,
	providerRepo domain.LLMProviderRepository,
	requirementRepo domain.RequirementRepository,
	replicaCleanupSvc domain.ReplicaCleanupService,
	conversationRepo domain.ConversationRecordRepository,
) *OpenCodeProcessor {
	return &OpenCodeProcessor{
		logger:            logger,
		hookManager:       hookManager,
		providerRepo:      providerRepo,
		requirementRepo:  requirementRepo,
		conversationRepo: conversationRepo,
		replicaCleanupSvc: replicaCleanupSvc,
	}
}

// resolveProvider 解析 LLM Provider
func (p *OpenCodeProcessor) resolveProvider(agent *domain.Agent) (*domain.LLMProvider, error) {
	if agent == nil {
		return nil, nil
	}

	providerID := agent.LLMProviderID()
	if providerID.String() == "" {
		return nil, nil
	}

	repo, ok := p.providerRepo.(domain.LLMProviderRepository)
	if !ok {
		return nil, nil
	}

	ctx := context.Background()
	return repo.FindByID(ctx, providerID)
}

// Process 处理消息（同步版本，内部调用流式版本）
func (p *OpenCodeProcessor) Process(
	ctx context.Context,
	msg *bus.InboundMessage,
	session *OpenCodeSession,
	agent *domain.Agent,
) (string, error) {
	var result string

	streamingErr := p.ProcessWithStreaming(ctx, msg, session, agent, &syncCallback{
		onText: func(text string) {
			result += text
		},
		onComplete: func(finalResult string) {
			result = finalResult
		},
	})

	if streamingErr != nil {
		return result, streamingErr
	}

	return result, nil
}

// syncCallback 同步回调，用于将流式结果同步化
type syncCallback struct {
	onText     func(text string)
	onComplete func(finalResult string)
}

func (c *syncCallback) OnThinking(thinking string) {}
func (c *syncCallback) OnToolCall(toolName string, input map[string]any) {}
func (c *syncCallback) OnToolResult(toolName string, result string) {}
func (c *syncCallback) OnText(text string) {
	if c.onText != nil {
		c.onText(text)
	}
}
func (c *syncCallback) OnComplete(finalResult string) {
	if c.onComplete != nil {
		c.onComplete(finalResult)
	}
}
func (c *syncCallback) GetFinalResult() string { return "" }

// ProcessWithStreaming 处理消息（流式版本）
func (p *OpenCodeProcessor) ProcessWithStreaming(
	ctx context.Context,
	msg *bus.InboundMessage,
	session *OpenCodeSession,
	agent *domain.Agent,
	callback StreamingCallback,
) error {
	sessionKey := msg.SessionKey()

	// 获取用户输入
	userInput := msg.Content
	if userInput == "" {
		return fmt.Errorf("user input is empty")
	}

	// 生成 trace ID
	traceID := fmt.Sprintf("oc_%d", time.Now().UnixNano())

	// 解析 Provider
	provider, err := p.resolveProvider(agent)
	if err != nil {
		p.logger.Warn("Failed to resolve provider",
			zap.String("session_key", sessionKey),
			zap.Error(err))
	}

	// 获取 CLI Session ID
	cliSessionID := ""
	if session != nil {
		cliSessionID = session.CliSessionID
	}

	// 执行流式处理
	result, _, err := p.queryOpenCodeStreaming(
		ctx, msg, userInput, cliSessionID, traceID, provider, agent, callback,
	)

	if err != nil {
		return fmt.Errorf("OpenCode query failed: %w", err)
	}

	p.logger.Info("OpenCode 处理完成",
		zap.String("session_key", sessionKey),
		zap.String("result_length", fmt.Sprintf("%d", len(result))))

	return nil
}

// queryOpenCodeStreaming 执行 OpenCode CLI 并流式处理输出
func (p *OpenCodeProcessor) queryOpenCodeStreaming(
	ctx context.Context,
	msg *bus.InboundMessage,
	userInput string,
	sessionID string,
	traceID string,
	provider *domain.LLMProvider,
	agent *domain.Agent,
	callback StreamingCallback,
) (string, *TokenUsage, error) {
	sessionKey := msg.SessionKey()

	// 获取配置
	config := agent.OpenCodeConfig()
	workDir := ""
	if config != nil && config.WorkDir != "" {
		workDir = config.WorkDir
	}

	// 构建命令
	args := buildCLIArgs(userInput, workDir, provider, config, sessionID)
	env := buildEnv(provider, config)

	p.logger.Info("开始 OpenCode 流式查询",
		zap.String("session_key", sessionKey),
		zap.String("session_id", sessionID),
		zap.Strings("args", args),
	)

	startTime := time.Now()

	// 创建命令
	cmd := exec.Command("opencode", args...)
	cmd.Env = env
	cmd.Dir = workDir

	// 创建 stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", nil, fmt.Errorf("创建 stdout pipe 失败: %w", err)
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		return "", nil, fmt.Errorf("启动 OpenCode 失败: %w", err)
	}

	// 创建工具钩子适配器
	toolAdapter := p.buildToolHookAdapter(msg, traceID, agent)

	// 解析流
	var cliSessionIDResult string
	var result string
	var tokenUsage *TokenUsage
	var mu sync.Mutex

	// 设置超时
	timeout := 600 // 10 分钟默认超时
	if config != nil && config.Timeout > 0 {
		timeout = config.Timeout
	}

	// 创建超时 context
	queryCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// 使用 goroutine 解析输出
	errChan := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var event OpenCodeEvent
			if err := json.Unmarshal(line, &event); err != nil {
				p.logger.Warn("Failed to parse JSON line",
					zap.String("line", string(line)),
					zap.Error(err))
				continue
			}

			// 处理事件
			switch event.Type {
			case "step_start":
				p.logger.Debug("OpenCode step started",
					zap.String("session_key", sessionKey))

			case "text":
				text := event.Part.Text
				if text != "" {
					mu.Lock()
					result += text
					mu.Unlock()
					callback.OnText(text)
				}

			case "thinking":
				thinking := event.Part.Thinking
				if thinking != "" {
					callback.OnThinking(thinking)
				}

			case "tool_use":
				toolName := event.Part.Tool
				input := event.Part.State.Input

				p.logger.Info("OpenCode 工具调用",
					zap.String("session_key", sessionKey),
					zap.String("tool", toolName))

				// 调用 PreToolCall hooks
				if toolAdapter != nil {
					toolAdapter.preToolUseAdapter(toolName, input)
				}

				// 通知工具调用
				callback.OnToolCall(toolName, input)

				// 如果工具已完成，处理结果
				if event.Part.State.Status == "completed" {
					output := event.Part.State.Output
					mu.Lock()
					result += output
					mu.Unlock()

					callback.OnToolResult(toolName, output)

					// 调用 PostToolCall hooks
					if toolAdapter != nil {
						toolAdapter.postToolUseAdapter(toolName, input, output, true)
					}
				} else if event.Part.State.Status == "error" {
					errMsg := ""
					if event.Part.State.Error != nil {
						errMsg = *event.Part.State.Error
					}
					callback.OnToolResult(toolName, fmt.Sprintf("Error: %s", errMsg))

					// 调用 PostToolCall hooks with error
					if toolAdapter != nil {
						toolAdapter.postToolUseAdapter(toolName, input, errMsg, false)
					}
				}

				// 保存 session ID
				if event.SessionID != "" {
					cliSessionIDResult = event.SessionID
				}

			case "step_finish":
				// 保存 token 使用量
				if event.Part.Tokens.Total > 0 {
					tokenUsage = &event.Part.Tokens
				}

				if event.Part.Reason == "stop" {
					p.logger.Info("OpenCode step finished (stop)",
						zap.String("session_key", sessionKey),
						zap.Duration("duration", time.Since(startTime)))
				}

			case "error":
				if event.Part.State.Error != nil {
					errMsg := *event.Part.State.Error
					mu.Lock()
					result += fmt.Sprintf("\n[Error: %s]", errMsg)
					mu.Unlock()
					p.logger.Error("OpenCode error",
						zap.String("session_key", sessionKey),
						zap.String("error", errMsg))
				}
			}
		}

		if err := scanner.Err(); err != nil {
			if queryCtx.Err() == nil {
				errChan <- fmt.Errorf("读取 stdout 失败: %w", err)
			}
		}

		close(errChan)
	}()

	// 等待完成或超时
	select {
	case <-queryCtx.Done():
		cmd.Process.Kill()
		return "", nil, queryCtx.Err()
	case err := <-errChan:
		cmd.Wait()
		if err != nil {
			return "", nil, err
		}
	}

	p.logger.Info("OpenCode 流式接收完成",
		zap.String("session_key", sessionKey),
		zap.Duration("duration", time.Since(startTime)),
		zap.String("result_length", fmt.Sprintf("%d", len(result))))

	callback.OnComplete(result)
	return cliSessionIDResult, tokenUsage, nil
}

// toolHookAdapter bridges OpenCode tool hooks to the domain hook system
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

// buildToolHookAdapter 构建工具钩子适配器
func (p *OpenCodeProcessor) buildToolHookAdapter(msg *bus.InboundMessage, traceID string, agent *domain.Agent) *toolHookAdapter {
	if p.hookManager == nil {
		return nil
	}

	sessionKey := msg.SessionKey()

	hookCtx := domain.NewHookContext(context.Background())
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

	return &toolHookAdapter{
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

// preToolUseAdapter 调用 PreToolCall hooks
func (a *toolHookAdapter) preToolUseAdapter(toolName string, input map[string]any) error {
	if a.hookManager == nil {
		return nil
	}

	callCtx := &domain.ToolCallContext{
		ToolName:     toolName,
		ToolInput:    input,
		SessionID:    a.sessionKey,
		TraceID:      a.traceID,
		SpanID:       "",
		ParentSpanID: "",
	}

	_, err := a.hookManager.PreToolCall(a.hookCtx, callCtx)
	if err != nil {
		a.logger.Error("OpenCode PreToolCall hook failed",
			zap.String("tool", toolName),
			zap.Error(err))
	}

	return nil
}

// postToolUseAdapter 调用 PostToolCall hooks
func (a *toolHookAdapter) postToolUseAdapter(toolName string, input map[string]any, output string, success bool) error {
	if a.hookManager == nil {
		return nil
	}

	callCtx := &domain.ToolCallContext{
		ToolName:     toolName,
		ToolInput:    input,
		SessionID:    a.sessionKey,
		TraceID:      a.traceID,
		SpanID:       "",
		ParentSpanID: "",
	}

	execResult := &domain.ToolExecutionResult{
		Success:  success,
		Duration: 0,
		Output:   output,
	}

	_, err := a.hookManager.PostToolCall(a.hookCtx, callCtx, execResult)
	if err != nil {
		a.logger.Error("OpenCode PostToolCall hook failed",
			zap.String("tool", toolName),
			zap.Error(err))
	}

	return nil
}
