package channel

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/weibh/taskmanager/infrastructure/hook"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

// contextKey 用于在 HookContext 中存储和获取数据
type contextKey string

const spanKey contextKey = "conversation_span"

// feishuStreamingCallback 实现流式回调，将部分结果发送到飞书（使用卡片格式）
type feishuStreamingCallback struct {
	bus         *bus.MessageBus
	logger      *zap.Logger
	inbound     *bus.InboundMessage
	traceID     string
	spanID      string
	hookManager *hook.Manager
	agentType   string
	mu          sync.Mutex
	finalResult string // 存储最终结果
}

// GetFinalResult 获取最终结果
func (c *feishuStreamingCallback) GetFinalResult() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.finalResult
}

func newFeishuStreamingCallback(bus *bus.MessageBus, logger *zap.Logger, inbound *bus.InboundMessage, traceID, spanID string, hookManager *hook.Manager, agentType string) *feishuStreamingCallback {
	return &feishuStreamingCallback{
		bus:         bus,
		logger:      logger,
		inbound:     inbound,
		traceID:     traceID,
		spanID:      spanID,
		hookManager: hookManager,
		agentType:   agentType,
	}
}

// displayName 返回用于消息标题展示的 Agent 名称
func (c *feishuStreamingCallback) displayName() string {
	if c.agentType == "OpenCodeAgent" {
		return "OpenCode"
	}
	return "Claude Code"
}

// escapeJSON 转义 JSON 字符串中的特殊字符
func escapeJSON(s string) string {
	result, _ := json.Marshal(s)
	return string(result)[1 : len(string(result))-1]
}

// buildThinkingCard 构建飞书思考过程卡片（复用 FeishuThinkingProcessHook 的格式）
func buildThinkingCard(title string, elements []map[string]interface{}) string {
	title = escapeJSON(title)

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
		"elements": elements,
	}

	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

func (c *feishuStreamingCallback) sendCard(title string, elements []map[string]interface{}) {
	if len(elements) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	cardContent := buildThinkingCard(title, elements)

	outMsg := &bus.OutboundMessage{
		Channel:  c.inbound.Channel,
		ChatID:   c.inbound.ChatID,
		Content:  cardContent,
		Metadata: c.buildMetadata(),
	}
	c.bus.PublishOutbound(outMsg)
}

func (c *feishuStreamingCallback) sendText(content string) {
	if content == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	outMsg := &bus.OutboundMessage{
		Channel:  c.inbound.Channel,
		ChatID:   c.inbound.ChatID,
		Content:  content,
		Metadata: c.buildMetadata(),
	}
	c.bus.PublishOutbound(outMsg)
}

func (c *feishuStreamingCallback) buildMetadata() map[string]any {
	m := make(map[string]any)
	if c.inbound.Metadata != nil {
		if msgID, ok := c.inbound.Metadata["message_id"].(string); ok {
			m["reply_to_message_id"] = msgID
		}
		if appID, ok := c.inbound.Metadata["app_id"].(string); ok {
			m["app_id"] = appID
		}
		if senderID, ok := c.inbound.Metadata["sender_id"].(string); ok {
			m["sender_id"] = senderID
		}
		if chatType, ok := c.inbound.Metadata["chat_type"].(string); ok {
			m["chat_type"] = chatType
		}
	}
	m["trace_id"] = c.traceID
	m["span_id"] = c.spanID
	m["msg_type"] = "interactive" // 标记为卡片消息
	return m
}

func (c *feishuStreamingCallback) OnThinking(thinking string) {
	// 发送思考过程卡片
	c.mu.Lock()
	defer c.mu.Unlock()

	// 判断是 p2p 还是群聊
	chatID := c.inbound.ChatID
	chatType := "p2p"
	if strings.HasPrefix(chatID, "oc_") {
		chatType = "group"
	}

	elements := []map[string]interface{}{
		{"tag": "markdown", "content": fmt.Sprintf("```\n%s\n```", escapeJSON(thinking))},
	}
	cardContent := buildThinkingCard(fmt.Sprintf("🤔 %s 思考过程", c.displayName()), elements)

	metadata := map[string]any{
		"msg_type":  "interactive",
		"chat_type": chatType,
	}

	msg := &bus.OutboundMessage{
		Channel:  c.inbound.Channel,
		ChatID:   chatID,
		Content:  cardContent,
		Metadata: metadata,
	}
	c.bus.PublishOutbound(msg)
}

func (c *feishuStreamingCallback) OnToolCall(toolName string, input map[string]any) {
	// 忽略工具调用，不发送中间卡片（由 hook 系统处理）
}

func (c *feishuStreamingCallback) OnToolResult(toolName string, result string) {
	// 忽略工具结果，不发送中间卡片（由 hook 系统处理）
}

func (c *feishuStreamingCallback) OnText(text string) {
	// 忽略中间文本，不立即发送
}

func (c *feishuStreamingCallback) OnComplete(finalResult string) {
	// 最终结果使用卡片格式发送
	// 思考内容已在 OnThinking 中单独发送，不需要过滤
	if finalResult == "" {
		return
	}

	// 存储最终结果
	c.mu.Lock()
	c.finalResult = finalResult
	c.mu.Unlock()

	if len(finalResult) > 2000 {
		finalResult = finalResult[:2000] + "..."
	}

	elements := []map[string]interface{}{
		{"tag": "markdown", "content": finalResult},
	}
	c.sendCard(fmt.Sprintf("🤖 %s 响应", c.displayName()), elements)

	c.logger.Info(fmt.Sprintf("%s 流式处理完成", c.displayName()),
		zap.String("trace_id", c.traceID),
		zap.Any("final_result_length", len(finalResult)),
	)
}
