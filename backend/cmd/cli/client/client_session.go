package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Session struct {
	SessionKey  string                 `json:"session_key"`
	UserCode    string                 `json:"user_code"`
	ChannelCode string                 `json:"channel_code"`
	AgentCode   string                 `json:"agent_code"`
	Status      string                 `json:"status"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   int64                  `json:"created_at"`
	UpdatedAt   int64                  `json:"updated_at"`
}

// ListUserSessions 获取用户会话列表
func (c *Client) ListUserSessions(ctx context.Context, userCode string) ([]Session, error) {
	path := "/sessions"
	if userCode != "" {
		path += "?user_code=" + userCode
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result []Session
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return result, nil
}

// DeleteSession 删除会话
func (c *Client) DeleteSession(ctx context.Context, sessionKey string) error {
	path := "/sessions?session_key=" + sessionKey
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return c.handleError(resp)
	}

	return nil
}

// ==================== User APIs ====================
