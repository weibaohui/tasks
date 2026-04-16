package channel

import (
	"context"
	"fmt"
	"strings"

	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

// CommandHandler 命令处理函数类型
type CommandHandler func(ctx context.Context, msg *bus.InboundMessage, args string) string

// CommandProcessor 命令处理器
type CommandProcessor struct {
	commands map[string]CommandHandler
	logger   *zap.Logger
}

// NewCommandProcessor 创建命令处理器
func NewCommandProcessor(logger *zap.Logger) *CommandProcessor {
	cp := &CommandProcessor{
		commands: make(map[string]CommandHandler),
		logger:   logger,
	}
	cp.registerDefaultCommands()
	return cp
}

// registerDefaultCommands 注册默认命令
func (cp *CommandProcessor) registerDefaultCommands() {
	// /help - 显示帮助信息
	cp.Register("/help", cp.handleHelp)
	// /status - 显示状态
	cp.Register("/status", cp.handleStatus)
	// /new - 创建新会话
	cp.Register("/new", cp.handleNew)
	// /stop - 停止当前会话
	cp.Register("/stop", cp.handleStop)
	// /clear - 清除会话历史
	cp.Register("/clear", cp.handleClear)
	// /models - 列出可用模型
	cp.Register("/models", cp.handleModels)
}

// Register 注册命令处理函数
func (cp *CommandProcessor) Register(command string, handler CommandHandler) {
	cp.commands[command] = handler
}

// IsCommand 检查消息是否是命令
func (cp *CommandProcessor) IsCommand(content string) bool {
	if !strings.HasPrefix(content, "/") {
		return false
	}
	parts := strings.Fields(content)
	if len(parts) == 0 {
		return false
	}
	command := strings.ToLower(parts[0])
	_, exists := cp.commands[command]
	return exists
}

// Process 处理命令
func (cp *CommandProcessor) Process(ctx context.Context, msg *bus.InboundMessage) string {
	content := strings.TrimSpace(msg.Content)
	parts := strings.Fields(content)
	if len(parts) == 0 {
		return "无效命令"
	}

	command := strings.ToLower(parts[0])
	handler, exists := cp.commands[command]
	if !exists {
		return fmt.Sprintf("未知命令: %s", command)
	}

	args := ""
	if len(parts) > 1 {
		args = strings.Join(parts[1:], " ")
	}

	cp.logger.Info("执行命令",
		zap.String("command", command),
		zap.String("args", args),
		zap.String("session_key", msg.SessionKey()),
	)

	return handler(ctx, msg, args)
}

// handleHelp 显示帮助信息
func (cp *CommandProcessor) handleHelp(ctx context.Context, msg *bus.InboundMessage, args string) string {
	help := `
可用命令:
/help - 显示帮助信息
/status - 显示当前状态
/new - 创建新会话
/stop - 停止当前 Claude Code 会话
/clear - 清除会话历史
/models - 列出支持的模型

直接在聊天中发送消息，Claude Code 会自动处理。
`
	return strings.TrimSpace(help)
}

// handleStatus 显示状态
func (cp *CommandProcessor) handleStatus(ctx context.Context, msg *bus.InboundMessage, args string) string {
	session := sessionManager.Get(msg.SessionKey())
	if session == nil {
		return "状态: 无活动会话"
	}

	sessionID := session.GetCliSessionID()
	if sessionID != "" {
		return fmt.Sprintf("状态: 会话活跃\n会话Key: %s\nCLI Session: %s", msg.SessionKey(), sessionID)
	}
	return fmt.Sprintf("状态: 会话活跃\n会话Key: %s\nCLI Session: 未创建", msg.SessionKey())
}

// handleNew 创建新会话
func (cp *CommandProcessor) handleNew(ctx context.Context, msg *bus.InboundMessage, args string) string {
	session := sessionManager.Get(msg.SessionKey())
	if session != nil {
		// 清除 CLI Session ID
		session.SetCliSessionID("")
	}
	return "已创建新会话，Claude Code 将开始新的对话上下文。"
}

// handleStop 停止当前 Claude Code 会话
func (cp *CommandProcessor) handleStop(ctx context.Context, msg *bus.InboundMessage, args string) string {
	session := sessionManager.Get(msg.SessionKey())
	if session != nil {
		cliSessionID := session.GetCliSessionID()
		if cliSessionID != "" {
			// 清除 CLI Session ID，下次将是新会话
			session.SetCliSessionID("")
			return fmt.Sprintf("已停止当前会话 (CLI Session: %s)", cliSessionID)
		}
	}
	return "没有活动的 Claude Code 会话需要停止。"
}

// handleClear 清除会话历史
func (cp *CommandProcessor) handleClear(ctx context.Context, msg *bus.InboundMessage, args string) string {
	session := sessionManager.Get(msg.SessionKey())
	if session != nil {
		// 清除会话（实际上是删除后重新创建）
		sessionManager.Delete(msg.SessionKey())
		return "会话历史已清除。"
	}
	return "没有会话历史需要清除。"
}

// handleModels 列出可用模型
func (cp *CommandProcessor) handleModels(ctx context.Context, msg *bus.InboundMessage, args string) string {
	return `
支持的模型:
- Claude 3.5 Sonnet (默认)
- Claude 3 Opus
- Claude 3 Haiku

模型配置在 Agent 设置中管理。
`
}

// sessionManager 是全局会话管理器引用（由 MessageProcessor 设置）
var sessionManager *SessionManager

// SetSessionManager 设置会话管理器引用
func SetSessionManager(sm *SessionManager) {
	sessionManager = sm
}