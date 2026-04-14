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

	// 保存 trace_id 到需求表
	if msg.Metadata != nil {
		if requirementIDStr, ok := msg.Metadata["requirement_id"].(string); ok && requirementIDStr != "" {
			requirementID := domain.NewRequirementID(requirementIDStr)
			requirement, err := p.requirementRepo.FindByID(ctx, requirementID)
			if err != nil {
				p.logger.Warn("查找需求失败，无法保存 trace_id", zap.Error(err))
			} else if requirement != nil {
				requirement.SetTraceID(traceID)
				if err := p.requirementRepo.Save(ctx, requirement); err != nil {
					p.logger.Warn("保存 trace_id 到需求表失败", zap.Error(err))
				} else {
					p.logger.Info("已保存 trace_id 到需求表",
						zap.String("requirement_id", requirementIDStr),
						zap.String("trace_id", traceID),
					)
				}
			}
		}
	}

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
	result, tokenUsage, err := p.queryOpenCodeStreaming(
		ctx, msg, userInput, cliSessionID, traceID, provider, agent, callback,
	)

	if err != nil {
		p.logger.Error("OpenCode 流式调用失败", zap.Error(err))
		p.triggerOpenCodeFinishedHook(ctx, msg, agent, false, "", nil)
		return fmt.Errorf("OpenCode query failed: %w", err)
	}

	totalTokens := 0
	if tokenUsage != nil {
		totalTokens = tokenUsage.Total
	}

	p.logger.Info("OpenCode 处理完成",
		zap.String("session_key", sessionKey),
		zap.String("result_length", fmt.Sprintf("%d", len(result))),
		zap.Int("total_tokens", totalTokens))

	p.triggerOpenCodeFinishedHook(ctx, msg, agent, true, result, tokenUsage)

	return nil
}

// streamContext holds mutable state while processing OpenCode CLI events.
type streamContext struct {
	p            *OpenCodeProcessor
	mu           sync.Mutex
	result       string
	tokenUsage   *TokenUsage
	cliSessionID string
	callback     StreamingCallback
	startTime    time.Time
	sessionKey   string
	toolAdapter  *hook.ToolHookBridge
}

func (sc *streamContext) appendResult(text string) {
	sc.mu.Lock()
	sc.result += text
	sc.mu.Unlock()
}

func (sc *streamContext) handleStepStart() {
	sc.p.logger.Debug("OpenCode step started",
		zap.String("session_key", sc.sessionKey))
}

func (sc *streamContext) handleTextEvent(text string) {
	if text == "" {
		return
	}
	sc.appendResult(text)
	sc.callback.OnText(text)
}

func (sc *streamContext) handleThinkingEvent(thinking string) {
	if thinking == "" {
		return
	}
	sc.callback.OnThinking(thinking)
}

func (sc *streamContext) handleToolUseEvent(event OpenCodeEvent) {
	toolName := event.Part.Tool
	input := event.Part.State.Input

	sc.p.logger.Info("OpenCode 工具调用",
		zap.String("session_key", sc.sessionKey),
		zap.String("tool", toolName))

	if sc.toolAdapter != nil {
		sc.toolAdapter.PreToolCall(toolName, input)
	}

	sc.callback.OnToolCall(toolName, input)

	switch event.Part.State.Status {
	case "completed":
		output := event.Part.State.Output
		sc.appendResult(output)
		sc.callback.OnToolResult(toolName, output)

		if sc.toolAdapter != nil {
			sc.toolAdapter.PostToolCall(toolName, input, output, true)
		}
	case "error":
		errMsg := ""
		if event.Part.State.Error != nil {
			errMsg = *event.Part.State.Error
		}
		sc.callback.OnToolResult(toolName, fmt.Sprintf("Error: %s", errMsg))

		if sc.toolAdapter != nil {
			sc.toolAdapter.PostToolCall(toolName, input, errMsg, false)
		}
	case "pending":
		// pending 状态下工具尚未完成，调用 PostToolCall 以关闭 span 防止资源泄漏
		if sc.toolAdapter != nil {
			sc.toolAdapter.PostToolCall(toolName, input, "", false)
		}
	default:
		// 未知状态：同样调用 PostToolCall 确保 span 关闭
		if sc.toolAdapter != nil {
			sc.toolAdapter.PostToolCall(toolName, input, "", false)
		}
	}

	if event.SessionID != "" {
		sc.cliSessionID = event.SessionID
	}
}

func (sc *streamContext) handleStepFinishEvent(tokens TokenUsage, reason string) {
	if tokens.Total > 0 {
		sc.tokenUsage = &tokens
	}

	if reason == "stop" {
		sc.p.logger.Info("OpenCode step finished (stop)",
			zap.String("session_key", sc.sessionKey),
			zap.Duration("duration", time.Since(sc.startTime)))
	}
}

func (sc *streamContext) handleErrorEvent(errMsg string) {
	sc.appendResult(fmt.Sprintf("\n[Error: %s]", errMsg))
	sc.p.logger.Error("OpenCode error",
		zap.String("session_key", sc.sessionKey),
		zap.String("error", errMsg))
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

	// 设置超时
	timeout := 600 // 10 分钟默认超时
	if config != nil && config.Timeout > 0 {
		timeout = config.Timeout
	}

	// 创建超时 context
	queryCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	sc := &streamContext{
		p:          p,
		callback:   callback,
		startTime:  startTime,
		sessionKey: sessionKey,
	}

	// 创建工具钩子适配器并设置 LLM Hook
	var toolAdapter *hook.ToolHookBridge
	var hookCtx *domain.HookContext
	var llmCallCtx *domain.LLMCallContext

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

		llmCallCtx = &domain.LLMCallContext{
			Prompt:    userInput,
			UserInput: userInput,
			Model:     "opencode",
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

		defer func() {
			resp := &domain.LLMResponse{Content: sc.result}
			if sc.tokenUsage != nil {
				resp.Usage = domain.Usage{
					PromptTokens:     sc.tokenUsage.Input,
					CompletionTokens: sc.tokenUsage.Output,
					TotalTokens:      sc.tokenUsage.Total,
				}
			}
			p.hookManager.PostLLMCall(hookCtx, llmCallCtx, resp)
			p.hookManager.OnToolExecutionComplete(hookCtx)
		}()

		toolAdapter = p.buildToolHookBridge(msg, traceID, agent, hookCtx)
		sc.toolAdapter = toolAdapter
	}

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
				sc.handleStepStart()
			case "text":
				sc.handleTextEvent(event.Part.Text)
			case "thinking":
				sc.handleThinkingEvent(event.Part.Thinking)
			case "tool_use":
				sc.handleToolUseEvent(event)
			case "step_finish":
				sc.handleStepFinishEvent(event.Part.Tokens, event.Part.Reason)
			case "error":
				if event.Part.State.Error != nil {
					sc.handleErrorEvent(*event.Part.State.Error)
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
		return sc.result, sc.tokenUsage, queryCtx.Err()
	case err := <-errChan:
		cmd.Wait()
		if err != nil {
			return sc.result, sc.tokenUsage, err
		}
	}

	p.logger.Info("OpenCode 流式接收完成",
		zap.String("session_key", sessionKey),
		zap.Duration("duration", time.Since(startTime)),
		zap.String("result_length", fmt.Sprintf("%d", len(sc.result))))

	callback.OnComplete(sc.result)
	return sc.result, sc.tokenUsage, nil
}

// buildToolHookBridge 构建通用的工具钩子桥接器
func (p *OpenCodeProcessor) buildToolHookBridge(msg *bus.InboundMessage, traceID string, agent *domain.Agent, hookCtx *domain.HookContext) *hook.ToolHookBridge {
	if p.hookManager == nil {
		return nil
	}

	sessionKey := msg.SessionKey()

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

	return &hook.ToolHookBridge{
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
