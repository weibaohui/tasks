package feishu

import (
	"encoding/json"
	"fmt"
	"strings"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

// isMarkdown 检测内容是否包含 markdown 格式
func isMarkdown(content string) bool {
	// 检测常见的 markdown 格式
	markdownIndicators := []string{
		"**",  // 粗体
		"`",   // 行内代码
		"```", // 代码块
		"- ",  // 列表
		"* ",  // 列表
		"##",  // 标题
		"---", // 分隔线
		"> ",  // 引用
	}
	for _, indicator := range markdownIndicators {
		if strings.Contains(content, indicator) {
			return true
		}
	}
	return false
}

// buildMarkdownCard 将 markdown 内容构建为飞书卡片格式
func buildMarkdownCard(content string) string {
	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"elements": []map[string]interface{}{
			{
				"tag":     "markdown",
				"content": content,
			},
		},
	}
	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// Send sends a message to Feishu
func (c *Channel) Send(msg *bus.OutboundMessage) error {
	if c.client == nil {
		return fmt.Errorf("Feishu client not initialized")
	}

	content := msg.Content

	// 跳过空消息
	if strings.TrimSpace(content) == "" {
		c.logger.Debug("Skipping empty/whitespace message")
		return nil
	}

	receiveIDType, receiveID := resolveReceiveTarget(msg)

	// 判断消息类型：text 或 interactive(卡片)
	msgType := "text"
	var contentStr string
	if msg.Metadata != nil {
		if mt, ok := msg.Metadata["msg_type"].(string); ok && mt == "interactive" {
			msgType = "interactive"
			// 卡片内容直接使用 Content（已经是 JSON 格式）
			contentStr = content
		}
	}

	// 如果是 markdown 内容，自动包装成卡片格式
	if msgType == "text" && isMarkdown(content) {
		msgType = "interactive"
		contentStr = buildMarkdownCard(content)
	}

	// 如果还不是卡片消息，使用文本格式
	if msgType == "text" {
		contentStr = fmt.Sprintf(`{"text":"%s"}`, escapeJSONString(content))
	}

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveIDType).
		Body(&larkim.CreateMessageReqBody{
			ReceiveId: &receiveID,
			MsgType:   ptrString(msgType),
			Content:   ptrRawMessage(contentStr),
		}).Build()

	resp, err := c.client.Im.V1.Message.Create(c.ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create Feishu message: %w", err)
	}

	if !resp.Success() {
		return fmt.Errorf("Feishu API error: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	if resp.Data != nil && resp.Data.MessageId != nil {
		c.logger.Debug("Message sent to Feishu",
			zap.String("receive_id", receiveID),
			zap.String("receive_id_type", receiveIDType),
			zap.String("message_id", *resp.Data.MessageId),
			zap.String("msg_type", msgType),
		)
	}

	return nil
}

func resolveReceiveTarget(msg *bus.OutboundMessage) (string, string) {
	if msg == nil {
		return "chat_id", ""
	}

	receiveIDType := "chat_id"
	receiveID := msg.ChatID

	if msg.Metadata == nil {
		return inferReceiveTargetByID(receiveIDType, receiveID)
	}

	chatType, _ := msg.Metadata["chat_type"].(string)
	senderID, _ := msg.Metadata["sender_id"].(string)
	originChatID, _ := msg.Metadata["chat_id"].(string)

	switch chatType {
	case "p2p":
		receiveIDType = "open_id"
		if senderID != "" {
			receiveID = senderID
		}
	case "group":
		receiveIDType = "chat_id"
		if originChatID != "" {
			receiveID = originChatID
		}
	}

	return inferReceiveTargetByID(receiveIDType, receiveID)
}

func inferReceiveTargetByID(receiveIDType, receiveID string) (string, string) {
	if strings.HasPrefix(receiveID, "ou_") {
		return "open_id", receiveID
	}
	if strings.HasPrefix(receiveID, "oc_") {
		return "chat_id", receiveID
	}
	return receiveIDType, receiveID
}

// SendWithReply sends a message to Feishu as a reply to another message
func (c *Channel) SendWithReply(msg *bus.OutboundMessage, replyToMessageID string) error {
	if c.client == nil {
		return fmt.Errorf("Feishu client not initialized")
	}

	content := msg.Content
	chatID := msg.ChatID

	msgType := "text"
	var contentStr string

	// 如果是 markdown 内容，自动包装成卡片格式
	if isMarkdown(content) {
		msgType = "interactive"
		contentStr = buildMarkdownCard(content)
	} else {
		contentStr = fmt.Sprintf(`{"text":"%s"}`, escapeJSONString(content))
	}

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType("chat_id").
		Body(&larkim.CreateMessageReqBody{
			ReceiveId: &chatID,
			MsgType:   ptrString(msgType),
			Content:   ptrRawMessage(contentStr),
		}).Build()

	resp, err := c.client.Im.V1.Message.Create(c.ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create Feishu reply message: %w", err)
	}

	if !resp.Success() {
		return fmt.Errorf("Feishu API error: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	return nil
}

// SendText sends a simple text message
func (c *Channel) SendText(chatID, text string) error {
	return c.Send(&bus.OutboundMessage{
		Channel: "feishu",
		ChatID:  chatID,
		Content: text,
	})
}

// escapeJSONString escapes special characters for JSON string
func escapeJSONString(s string) string {
	result, _ := json.Marshal(s)
	return string(result)[1 : len(string(result))-1]
}

// ptrString returns a pointer to a string
func ptrString(s string) *string {
	return &s
}

// ptrRawMessage returns a pointer to a raw message
func ptrRawMessage(s string) *string {
	return &s
}
