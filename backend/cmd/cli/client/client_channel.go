package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Channel struct {
	ID          string                 `json:"id"`
	ChannelCode string                 `json:"channel_code"`
	UserCode    string                 `json:"user_code"`
	AgentCode   string                 `json:"agent_code"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	IsActive    bool                   `json:"is_active"`
	AllowFrom   []string               `json:"allow_from"`
	Config      map[string]interface{} `json:"config"`
	CreatedAt   int64                  `json:"created_at"`
	UpdatedAt   int64                  `json:"updated_at"`
}

// ListChannels 获取渠道列表
func (c *Client) ListChannels(ctx context.Context, userCode string) ([]Channel, error) {
	path := "/channels"
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

	var result []Channel
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return result, nil
}

// CreateChannelAPIRequest 创建渠道请求
type CreateChannelAPIRequest struct {
	UserCode  string                 `json:"user_code"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Config    map[string]interface{} `json:"config"`
	AllowFrom []string               `json:"allow_from"`
	AgentCode string                 `json:"agent_code"`
}

// CreateChannel 创建渠道
func (c *Client) CreateChannel(ctx context.Context, req CreateChannelAPIRequest) (*Channel, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/channels", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleError(resp)
	}

	var result Channel
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// UpdateChannelAPIRequest 更新渠道请求
type UpdateChannelAPIRequest struct {
	Name      *string                 `json:"name,omitempty"`
	Config    *map[string]interface{} `json:"config,omitempty"`
	AllowFrom *[]string               `json:"allow_from,omitempty"`
	IsActive  *bool                   `json:"is_active,omitempty"`
	AgentCode *string                 `json:"agent_code,omitempty"`
}

// UpdateChannel 更新渠道
func (c *Client) UpdateChannel(ctx context.Context, id string, req UpdateChannelAPIRequest) (*Channel, error) {
	path := fmt.Sprintf("/channels?id=%s", id)
	resp, err := c.doRequest(ctx, http.MethodPut, path, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result Channel
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// DeleteChannel 删除渠道
func (c *Client) DeleteChannel(ctx context.Context, id string) error {
	path := "/channels?id=" + id
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

// ==================== Provider APIs ====================
