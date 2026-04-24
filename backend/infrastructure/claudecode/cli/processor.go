package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/weibh/taskmanager/domain"
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

// streamContext holds mutable state while processing CLI events
type streamContext struct {
	result       string
	tokenUsage   *TokenUsage
	cliSessionID string
	lastToolName string
	mu           sync.Mutex
	err          error
}

// Processor 处理 Claude Code CLI 调用
type Processor struct {
	logger *zap.Logger
}

// NewProcessor 创建 Processor
func NewProcessor(logger *zap.Logger) *Processor {
	return &Processor{logger: logger}
}

// QueryStreaming 执行 claude CLI 并流式处理输出
func (p *Processor) QueryStreaming(
	ctx context.Context,
	msg *bus.InboundMessage,
	userInput, cliSessionID, traceID string,
	provider *domain.LLMProvider,
	config *domain.ClaudeCodeConfig,
	callback StreamingCallback,
) (string, *TokenUsage, string, error) {
	sessionKey := msg.SessionKey()

	// 构建 CLI 参数
	args := buildCLIArgs(userInput, cliSessionID, provider, config)
	env := buildEnv(provider, config)

	p.logger.Info("开始 Claude CLI 流式查询",
		zap.String("session_key", sessionKey),
		zap.Strings("args", args),
	)

	startTime := time.Now()

	// 创建命令
	cmd := exec.Command("claude", args...)
	cmd.Env = env

	// 设置工作目录
	if config != nil && config.Cwd != "" {
		cmd.Dir = config.Cwd
	}

	// 创建 stdout / stderr pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", nil, "", fmt.Errorf("创建 stdout pipe 失败: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", nil, "", fmt.Errorf("创建 stderr pipe 失败: %w", err)
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		return "", nil, "", fmt.Errorf("启动 Claude CLI 失败: %w", err)
	}

	// 在后台读取 stderr
	var stderrBuilder strings.Builder
	go func() {
		io.Copy(&stderrBuilder, stderr)
	}()

	// 设置超时
	timeout := 3600 // 1 小时默认超时
	if config != nil && config.Timeout > 0 {
		timeout = config.Timeout
	}

	// 创建超时 context
	queryCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	sc := &streamContext{}

	// 使用 goroutine 解析输出
	scannerDone := make(chan struct{})
	go func() {
		defer close(scannerDone)
		scanner := bufio.NewScanner(stdout)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var event StreamEvent
			if err := json.Unmarshal(line, &event); err != nil {
				p.logger.Warn("Failed to parse JSON line",
					zap.String("line", string(line)),
					zap.Error(err))
				continue
			}

			p.handleEvent(sessionKey, &event, sc, callback)
		}

		if err := scanner.Err(); err != nil {
			if queryCtx.Err() == nil {
				sc.mu.Lock()
				sc.err = fmt.Errorf("读取 stdout 失败: %w", err)
				sc.mu.Unlock()
			}
		}
	}()

	// 在后台等待进程结束
	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	// 等待完成或超时
	select {
	case <-queryCtx.Done():
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		go func() { <-waitDone }()
		return sc.result, sc.tokenUsage, sc.cliSessionID, queryCtx.Err()
	case waitErr := <-waitDone:
		<-scannerDone
		stderrStr := strings.TrimSpace(stderrBuilder.String())
		if stderrStr != "" {
			p.logger.Warn("Claude CLI stderr",
				zap.String("session_key", sessionKey),
				zap.String("stderr", stderrStr))
		}
		if waitErr != nil {
			if stderrStr != "" {
				return sc.result, sc.tokenUsage, sc.cliSessionID, fmt.Errorf("Claude CLI 进程退出失败: %w (stderr: %s)", waitErr, stderrStr)
			}
			return sc.result, sc.tokenUsage, sc.cliSessionID, fmt.Errorf("Claude CLI 进程退出失败: %w", waitErr)
		}
		if sc.err != nil {
			return sc.result, sc.tokenUsage, sc.cliSessionID, fmt.Errorf("Claude CLI 执行中出现错误: %w", sc.err)
		}
	}

	p.logger.Info("Claude CLI 流式接收完成",
		zap.String("session_key", sessionKey),
		zap.Duration("duration", time.Since(startTime)),
		zap.String("result_length", fmt.Sprintf("%d", len(sc.result))))

	callback.OnComplete(sc.result)
	return sc.result, sc.tokenUsage, sc.cliSessionID, nil
}

// handleEvent 处理单个 stream-json 事件
func (p *Processor) handleEvent(sessionKey string, event *StreamEvent, sc *streamContext, callback StreamingCallback) {
	switch event.Type {
	case "system":
		// init 消息，包含 session_id
		if event.SessionID != "" {
			sc.cliSessionID = event.SessionID
		}

	case "assistant":
		if event.Message == nil {
			return
		}
		for _, block := range event.Message.Content {
			switch block.Type {
			case "thinking":
				if block.Thinking != "" {
					callback.OnThinking(block.Thinking)
				}
			case "text":
				if block.Text != "" {
					sc.mu.Lock()
					sc.result += block.Text
					sc.mu.Unlock()
					callback.OnText(block.Text)
				}
			case "tool_use":
				toolName := block.Name
				var input map[string]any
				if block.Input != nil {
					if m, ok := block.Input.(map[string]any); ok {
						input = m
					}
				}
				sc.mu.Lock()
				sc.lastToolName = toolName
				sc.mu.Unlock()
				callback.OnToolCall(toolName, input)
			case "tool_result":
				var content string
				if block.Content != nil {
					content = fmt.Sprintf("%v", block.Content)
				}
				sc.mu.Lock()
				sc.result += content
				toolName := sc.lastToolName
				sc.mu.Unlock()
				callback.OnToolResult(toolName, content)
			}
		}

	case "result":
		if event.SessionID != "" {
			sc.cliSessionID = event.SessionID
		}
		if event.Result != "" {
			sc.mu.Lock()
			sc.result = event.Result
			sc.mu.Unlock()
		}
		if event.Usage != nil {
			sc.tokenUsage = &TokenUsage{
				InputTokens:               event.Usage.InputTokens,
				OutputTokens:             event.Usage.OutputTokens,
				CacheCreationInputTokens: event.Usage.CacheCreationInputTokens,
				CacheReadInputTokens:     event.Usage.CacheReadInputTokens,
			}
			sc.tokenUsage.Total = sc.tokenUsage.InputTokens + sc.tokenUsage.OutputTokens +
				sc.tokenUsage.CacheCreationInputTokens + sc.tokenUsage.CacheReadInputTokens
		}
		if event.IsError {
			sc.mu.Lock()
			sc.err = fmt.Errorf("Claude CLI error: %s", event.Result)
			sc.mu.Unlock()
		}

	case "error":
		errMsg := ""
		if event.Error != nil {
			if msg, ok := event.Error.Data["message"].(string); ok {
				errMsg = msg
			} else {
				errMsg = event.Error.Name
			}
		}
		sc.mu.Lock()
		sc.err = fmt.Errorf("%s", errMsg)
		sc.mu.Unlock()
		p.logger.Error("Claude CLI stream error",
			zap.String("session_key", sessionKey),
			zap.Any("error", event.Error))
	}
}
