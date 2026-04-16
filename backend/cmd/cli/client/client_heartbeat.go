package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Heartbeat struct {
	ID              string `json:"id"`
	ProjectID       string `json:"project_id"`
	Name            string `json:"name"`
	Enabled         bool   `json:"enabled"`
	IntervalMinutes int    `json:"interval_minutes"`
	MDContent       string `json:"md_content"`
	AgentCode       string `json:"agent_code"`
	RequirementType string `json:"requirement_type"`
	SortOrder       int    `json:"sort_order"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}

type CreateHeartbeatRequest struct {
	ProjectID       string `json:"project_id"`
	Name            string `json:"name"`
	IntervalMinutes int    `json:"interval_minutes"`
	MDContent       string `json:"md_content"`
	AgentCode       string `json:"agent_code"`
	RequirementType string `json:"requirement_type"`
}

type UpdateHeartbeatRequest struct {
	Name            string `json:"name"`
	IntervalMinutes int    `json:"interval_minutes"`
	MDContent       string `json:"md_content"`
	AgentCode       string `json:"agent_code"`
	RequirementType string `json:"requirement_type"`
	Enabled         bool   `json:"enabled"`
}

// ListHeartbeats 获取项目的心跳列表
func (c *Client) ListHeartbeats(ctx context.Context, projectID string) ([]Heartbeat, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/heartbeats?project_id="+projectID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result []Heartbeat
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}
	return result, nil
}

// CreateHeartbeat 创建心跳
func (c *Client) CreateHeartbeat(ctx context.Context, req CreateHeartbeatRequest) (*Heartbeat, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/heartbeats", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, c.handleError(resp)
	}

	var result Heartbeat
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}
	return &result, nil
}

// GetHeartbeat 获取心跳详情
func (c *Client) GetHeartbeat(ctx context.Context, id string) (*Heartbeat, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/heartbeats/"+id, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result Heartbeat
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}
	return &result, nil
}

// UpdateHeartbeat 更新心跳
func (c *Client) UpdateHeartbeat(ctx context.Context, id string, req UpdateHeartbeatRequest) (*Heartbeat, error) {
	resp, err := c.doRequest(ctx, http.MethodPut, "/heartbeats/"+id, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result Heartbeat
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}
	return &result, nil
}

// DeleteHeartbeat 删除心跳
func (c *Client) DeleteHeartbeat(ctx context.Context, id string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, "/heartbeats/"+id, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return c.handleError(resp)
	}
	return nil
}

// TriggerHeartbeat 手动触发心跳
func (c *Client) TriggerHeartbeat(ctx context.Context, id string) error {
	resp, err := c.doRequest(ctx, http.MethodPost, "/heartbeats/"+id+"/trigger", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.handleError(resp)
	}
	return nil
}
