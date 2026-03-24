package feishu

import (
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"go.uber.org/zap"
)

// addReactionAndSave adds a reaction emoji to a message and saves to cache
func (c *Channel) addReactionAndSave(messageID, emojiType string) {
	if c.client == nil || messageID == "" {
		return
	}

	req := larkim.NewCreateMessageReactionReqBuilder().
		MessageId(messageID).
		Body(larkim.NewCreateMessageReactionReqBodyBuilder().
			ReactionType(larkim.NewEmojiBuilder().
				EmojiType(emojiType).
				Build()).
			Build()).
		Build()

	resp, err := c.client.Im.V1.MessageReaction.Create(c.ctx, req)
	if err != nil {
		c.logger.Debug("添加飞书反应表情失败", zap.Error(err))
		return
	}

	if !resp.Success() {
		c.logger.Debug("添加飞书反应表情失败",
			zap.Int("code", resp.Code),
			zap.String("msg", resp.Msg),
		)
		return
	}

	// Save reaction_id to cache
	if resp.Data != nil && resp.Data.ReactionId != nil {
		c.reactionMu.Lock()
		c.reactionCache[messageID] = &reactionInfo{
			messageID:  messageID,
			reactionID: *resp.Data.ReactionId,
		}
		c.reactionMu.Unlock()
		c.logger.Debug("已保存飞书反应表情",
			zap.String("message_id", messageID),
			zap.String("reaction_id", *resp.Data.ReactionId),
		)
	}
}

// deleteReactionFromCache deletes a reaction from cache
func (c *Channel) deleteReactionFromCache(messageID string) {
	if c.client == nil || messageID == "" {
		return
	}

	c.reactionMu.RLock()
	info, exists := c.reactionCache[messageID]
	c.reactionMu.RUnlock()

	if !exists {
		c.logger.Debug("未找到要删除的反应表情",
			zap.String("message_id", messageID),
		)
		return
	}

	req := larkim.NewDeleteMessageReactionReqBuilder().
		MessageId(info.messageID).
		ReactionId(info.reactionID).
		Build()

	resp, err := c.client.Im.V1.MessageReaction.Delete(c.ctx, req)
	if err != nil {
		c.logger.Debug("删除飞书反应表情失败", zap.Error(err))
		return
	}

	if !resp.Success() {
		c.logger.Debug("删除飞书反应表情失败",
			zap.Int("code", resp.Code),
			zap.String("msg", resp.Msg),
		)
		return
	}

	// Remove from cache
	c.reactionMu.Lock()
	delete(c.reactionCache, messageID)
	c.reactionMu.Unlock()

	c.logger.Debug("已删除飞书反应表情",
		zap.String("message_id", messageID),
		zap.String("reaction_id", info.reactionID),
	)
}
