package feishu

import (
	"encoding/json"
	"fmt"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/weibh/taskmanager/pkg/bus"
	"go.uber.org/zap"
)

// Send sends a message to Feishu
func (c *Channel) Send(msg *bus.OutboundMessage) error {
	if c.client == nil {
		return fmt.Errorf("Feishu client not initialized")
	}

	content := msg.Content
	chatID := msg.ChatID

	// Determine receive_id type based on chat_type
	// For p2p (person-to-person) chats, use open_id; for group chats, use chat_id
	receiveIDType := "chat_id"
	receiveID := chatID

	if msg.Metadata != nil {
		if chatType, ok := msg.Metadata["chat_type"].(string); ok && chatType == "p2p" {
			// For p2p chats, use sender's open_id as receive_id
			receiveIDType = "open_id"
			if senderID, ok := msg.Metadata["sender_id"].(string); ok && senderID != "" {
				receiveID = senderID
			}
		}
	}

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveIDType).
		Body(&larkim.CreateMessageReqBody{
			ReceiveId: &receiveID,
			MsgType:   ptrString("text"),
			Content:   ptrRawMessage(fmt.Sprintf(`{"text":"%s"}`, escapeJSONString(content))),
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
		)
	}

	return nil
}

// SendWithReply sends a message to Feishu as a reply to another message
func (c *Channel) SendWithReply(msg *bus.OutboundMessage, replyToMessageID string) error {
	if c.client == nil {
		return fmt.Errorf("Feishu client not initialized")
	}

	content := msg.Content
	chatID := msg.ChatID

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType("chat_id").
		Body(&larkim.CreateMessageReqBody{
			ReceiveId:  &chatID,
			MsgType:    ptrString("text"),
			Content:    ptrRawMessage(fmt.Sprintf(`{"text":"%s"}`, escapeJSONString(content))),
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
