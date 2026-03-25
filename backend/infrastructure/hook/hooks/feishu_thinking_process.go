/**
 * FeishuThinkingProcessHook - 飞书思考过程 Hook
 * 当 Agent 开启 enable_thinking_process 时，将 LLM 思考过程实时发送到飞书
 */
package hooks

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

// FeishuThinkingProcessHook 飞书思考过程 Hook
type FeishuThinkingProcessHook struct {
	*domain.BaseHook
	messageBus    *bus.MessageBus
	logger        *zap.Logger
	sessionCache  map[string]*sessionInfo
	mu            sync.RWMutex
}

// sessionInfo 会话信息缓存
type sessionInfo struct {
	SessionKey            string
	ChatID                string
	Channel               string
	UserCode              string
	AgentCode             string
	ChannelCode           string
	EnableThinkingProcess bool
	UpdatedAt             time.Time
}

// NewFeishuThinkingProcessHook 创建飞书思考过程 Hook
func NewFeishuThinkingProcessHook(
	messageBus *bus.MessageBus,
	logger *zap.Logger,
) *FeishuThinkingProcessHook {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &FeishuThinkingProcessHook{
		BaseHook:     domain.NewBaseHook("feishu_thinking_process", 60, domain.HookTypeLLM),
		messageBus:   messageBus,
		logger:       logger,
		sessionCache: make(map[string]*sessionInfo),
	}
}

// PreLLMCall 记录 LLM 调用开始
func (h *FeishuThinkingProcessHook) PreLLMCall(ctx *domain.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
	// 更新会话缓存
	h.updateSessionCache(ctx, callCtx)

	// 检查是否开启思考过程
	if !h.isThinkingProcessEnabled(ctx, callCtx) {
		return callCtx, nil
	}

	// 发送开始思考消息
	h.sendThinkingMessage(ctx, "🤔 开始思考", "**开始思考**...")

	return callCtx, nil
}

// PostLLMCall 记录 LLM 响应
func (h *FeishuThinkingProcessHook) PostLLMCall(ctx *domain.HookContext, callCtx *domain.LLMCallContext, resp *domain.LLMResponse) (*domain.LLMResponse, error) {
	// 检查是否开启思考过程
	if !h.isThinkingProcessEnabled(ctx, callCtx) {
		return resp, nil
	}

	// 如果包含工具调用，显示工具调用决定
	if resp.ContainsToolCalls {
		toolNames := h.extractToolNames(resp.RawResponse)
		if len(toolNames) > 0 {
			msg := fmt.Sprintf("**决策**: 调用 %s", strings.Join(toolNames, ", "))
			h.sendThinkingMessage(ctx, "🤖 工具决策", msg)
		}
	}

	return resp, nil
}

// PreToolCall 记录工具调用开始
func (h *FeishuThinkingProcessHook) PreToolCall(ctx *domain.HookContext, callCtx *domain.ToolCallContext) (*domain.ToolCallContext, error) {
	// 检查是否开启思考过程
	if !h.isThinkingProcessEnabled(ctx, nil) {
		return callCtx, nil
	}

	// 格式化工具参数
	argsJSON, _ := json.Marshal(callCtx.ToolInput)
	args := string(argsJSON)
	if len(args) > 200 {
		args = args[:200] + "..."
	}

	msg := fmt.Sprintf("```json\n%s\n```", args)
	title := fmt.Sprintf("🔧 执行工具: %s", callCtx.ToolName)
	h.sendThinkingMessage(ctx, title, msg)

	return callCtx, nil
}

// PostToolCall 记录工具执行完成
func (h *FeishuThinkingProcessHook) PostToolCall(ctx *domain.HookContext, callCtx *domain.ToolCallContext, result *domain.ToolExecutionResult) (*domain.ToolExecutionResult, error) {
	// 检查是否开启思考过程
	if !h.isThinkingProcessEnabled(ctx, nil) {
		return result, nil
	}

	output := ""
	if result.Success {
		if out, ok := result.Output.(string); ok {
			output = out
		} else {
			output = fmt.Sprintf("%v", result.Output)
		}
	} else {
		if result.Error != nil {
			output = result.Error.Error()
		} else {
			output = "unknown error"
		}
	}

	// 截断输出
	if len(output) > 300 {
		output = output[:300] + "..."
	}
	output = strings.TrimSpace(output)
	if output == "" {
		output = "(无输出)"
	}

	statusIcon := "✅"
	if !result.Success {
		statusIcon = "❌"
	}
	msg := fmt.Sprintf("**耗时**: %dms\n```\n%s\n```",
		result.Duration.Milliseconds(), output)
	title := fmt.Sprintf("%s 工具完成: %s", statusIcon, callCtx.ToolName)
	h.sendThinkingMessage(ctx, title, msg)

	return result, nil
}

// OnToolError 记录工具执行错误
func (h *FeishuThinkingProcessHook) OnToolError(ctx *domain.HookContext, callCtx *domain.ToolCallContext, err error) (*domain.ToolExecutionResult, error) {
	// 检查是否开启思考过程
	if !h.isThinkingProcessEnabled(ctx, nil) {
		return &domain.ToolExecutionResult{Success: false, Error: err}, nil
	}

	msg := fmt.Sprintf("```\n%s\n```", err.Error())
	title := fmt.Sprintf("❌ 工具错误: %s", callCtx.ToolName)
	h.sendThinkingMessage(ctx, title, msg)

	return &domain.ToolExecutionResult{Success: false, Error: err}, nil
}

// updateSessionCache 更新会话缓存
func (h *FeishuThinkingProcessHook) updateSessionCache(ctx *domain.HookContext, callCtx *domain.LLMCallContext) {
	if callCtx == nil || callCtx.Metadata == nil {
		return
	}

	sessionKey := callCtx.SessionID
	if sessionKey == "" {
		sessionKey = callCtx.Metadata["session_key"]
	}
	if sessionKey == "" {
		return
	}

	// 从 metadata 中提取会话信息
	info := &sessionInfo{
		SessionKey:  sessionKey,
		ChatID:      callCtx.Metadata["chat_id"],
		Channel:     callCtx.Metadata["channel_type"],
		UserCode:    callCtx.Metadata["user_code"],
		AgentCode:   callCtx.Metadata["agent_code"],
		ChannelCode: callCtx.Metadata["channel_code"],
		UpdatedAt:   time.Now(),
	}

	// 解析 enable_thinking_process
	if v := callCtx.Metadata["enable_thinking_process"]; v == "true" {
		info.EnableThinkingProcess = true
	}

	h.mu.Lock()
	h.sessionCache[sessionKey] = info
	h.mu.Unlock()
}

// isThinkingProcessEnabled 检查是否开启思考过程
func (h *FeishuThinkingProcessHook) isThinkingProcessEnabled(ctx *domain.HookContext, callCtx *domain.LLMCallContext) bool {
	// 优先从 callCtx.Metadata 检查
	if callCtx != nil && callCtx.Metadata != nil {
		v := callCtx.Metadata["enable_thinking_process"]
		if v == "true" {
			return true
		}
		// 如果明确设置了 false，直接返回 false
		if v == "false" {
			return false
		}
	}

	// 其次从 sessionCache 检查
	var sessionKey string
	if callCtx != nil {
		sessionKey = callCtx.SessionID
	}
	if sessionKey == "" && ctx != nil {
		sessionKey = ctx.GetMetadata("session_key")
	}

	if sessionKey != "" {
		h.mu.RLock()
		info, exists := h.sessionCache[sessionKey]
		h.mu.RUnlock()
		if exists && info.EnableThinkingProcess {
			return true
		}
	}

	return false
}

// getSessionInfo 获取会话信息
func (h *FeishuThinkingProcessHook) getSessionInfo(ctx *domain.HookContext) *sessionInfo {
	if ctx == nil {
		return nil
	}

	sessionKey := ctx.GetMetadata("session_key")
	if sessionKey == "" {
		return nil
	}

	h.mu.RLock()
	info, exists := h.sessionCache[sessionKey]
	h.mu.RUnlock()

	if !exists {
		return nil
	}

	// 检查缓存是否过期（30分钟）
	if time.Since(info.UpdatedAt) > 30*time.Minute {
		return nil
	}

	return info
}

// sendThinkingMessage 发送思考过程消息到飞书（使用卡片格式）
func (h *FeishuThinkingProcessHook) sendThinkingMessage(ctx *domain.HookContext, title, content string) {
	if h.messageBus == nil || ctx == nil {
		return
	}

	info := h.getSessionInfo(ctx)
	if info == nil {
		// 尝试从 context metadata 获取基本信息
		chatID := ctx.GetMetadata("chat_id")
		channel := ctx.GetMetadata("channel_type")
		if chatID == "" || channel == "" {
			h.logger.Debug("无法获取会话信息，跳过发送思考消息")
			return
		}
		info = &sessionInfo{
			ChatID:  chatID,
			Channel: channel,
		}
	}

	// 只发送到飞书渠道
	if info.Channel != "feishu" && info.Channel != "lark" {
		return
	}

	chatID := info.ChatID
	if chatID == "" {
		chatID = ctx.GetMetadata("chat_id")
	}
	if chatID == "" {
		h.logger.Debug("无法获取 ChatID，跳过发送思考消息")
		return
	}

	// 构建卡片内容
	cardContent := h.buildThinkingCard(title, content)

	contentPreview := content
	if len(contentPreview) > 100 {
		contentPreview = contentPreview[:100] + "..."
	}
	h.logger.Debug("[ThinkingProcess] 发送思考卡片",
		zap.String("channel", info.Channel),
		zap.String("chat_id", chatID),
		zap.String("title", title),
		zap.String("content_preview", contentPreview),
	)

	msg := &bus.OutboundMessage{
		Channel: info.Channel,
		ChatID:  chatID,
		Content: cardContent,
		Metadata: map[string]any{
			"type":            "thinking_process",
			"msg_type":        "interactive", // 标记为卡片消息
			"agent_code":      info.AgentCode,
			"user_code":       info.UserCode,
			"channel_code":    info.ChannelCode,
			"timestamp":       time.Now().Unix(),
		},
	}

	// 异步发送，避免阻塞主流程
	go func() {
		defer func() {
			if r := recover(); r != nil {
				h.logger.Error("发送思考过程消息 panic",
					zap.Any("recover", r),
				)
			}
		}()
		h.messageBus.PublishOutbound(msg)
	}()
}

// buildThinkingCard 构建飞书思考过程卡片
func (h *FeishuThinkingProcessHook) buildThinkingCard(title, content string) string {
	// Markdown 代码块不需要转义，lark_md 会直接渲染
	// 只有普通文本需要转义
	if !strings.HasPrefix(content, "```") {
		content = escapeJSON(content)
	}
	title = escapeJSON(title)

	// 构建飞书交互式卡片
	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": "blue",
			"title": map[string]interface{}{
				"content": title,
				"tag":     "plain_text",
			},
		},
		"elements": []map[string]interface{}{
			{
				"tag": "div",
				"text": map[string]interface{}{
					"content": content,
					"tag":     "lark_md",
				},
			},
		},
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// escapeJSON 转义 JSON 字符串中的特殊字符
func escapeJSON(s string) string {
	result, _ := json.Marshal(s)
	return string(result)[1 : len(string(result))-1]
}

// extractToolNames 从 RawResponse 提取工具名称
func (h *FeishuThinkingProcessHook) extractToolNames(rawResponse string) []string {
	if rawResponse == "" {
		return nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(rawResponse), &data); err != nil {
		return nil
	}

	toolCalls, ok := data["tool_calls"].([]interface{})
	if !ok {
		return nil
	}

	var names []string
	for _, tc := range toolCalls {
		if tcMap, ok := tc.(map[string]interface{}); ok {
			if fn, ok := tcMap["function"].(map[string]interface{}); ok {
				if name, ok := fn["name"].(string); ok && name != "" {
					names = append(names, "`"+name+"`")
				}
			}
		}
	}

	return names
}
