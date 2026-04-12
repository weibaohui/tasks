package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Project struct {
	ID                       string   `json:"id"`
	Name                     string   `json:"name"`
	GitRepoURL               string   `json:"git_repo_url"`
	DefaultBranch            string   `json:"default_branch"`
	InitSteps                []string `json:"init_steps"`
	HeartbeatEnabled         bool     `json:"heartbeat_enabled"`
	HeartbeatIntervalMinutes int      `json:"heartbeat_interval_minutes"`
	HeartbeatMDContent       string   `json:"heartbeat_md_content"`
	AgentCode                string   `json:"agent_code"`
	DispatchChannelCode      string   `json:"dispatch_channel_code"`
	DispatchSessionKey       string   `json:"dispatch_session_key"`
	CreatedAt                int64    `json:"created_at"`
	UpdatedAt                int64    `json:"updated_at"`
}

// CreateProjectRequest 创建项目请求
type CreateProjectRequest struct {
	Name          string   `json:"name"`
	GitRepoURL    string   `json:"git_repo_url"`
	DefaultBranch string   `json:"default_branch"`
	InitSteps     []string `json:"init_steps"`
}

// ListProjects 获取项目列表
func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/projects", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result []Project
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return result, nil
}

// CreateProject 创建项目
func (c *Client) CreateProject(ctx context.Context, req CreateProjectRequest) (*Project, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/projects", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, c.handleError(resp)
	}

	var result Project
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// GetProject 获取项目详情
func (c *Client) GetProject(ctx context.Context, id string) (*Project, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/projects?id="+id, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result Project
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// UpdateProjectRequest 更新项目请求
type UpdateProjectRequest struct {
	ID                       string   `json:"id"`
	Name                     string   `json:"name,omitempty"`
	GitRepoURL               string   `json:"git_repo_url,omitempty"`
	DefaultBranch            string   `json:"default_branch,omitempty"`
	InitSteps                []string `json:"init_steps,omitempty"`
	HeartbeatEnabled         *bool    `json:"heartbeat_enabled,omitempty"`
	HeartbeatIntervalMinutes *int     `json:"heartbeat_interval_minutes,omitempty"`
	HeartbeatMDContent       *string  `json:"heartbeat_md_content,omitempty"`
	AgentCode                *string  `json:"agent_code,omitempty"`
	DispatchChannelCode      *string  `json:"dispatch_channel_code,omitempty"`
	DispatchSessionKey       *string  `json:"dispatch_session_key,omitempty"`
}

// UpdateProject 更新项目
func (c *Client) UpdateProject(ctx context.Context, req UpdateProjectRequest) (*Project, error) {
	resp, err := c.doRequest(ctx, http.MethodPut, "/projects", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result Project
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// UpdateProjectHeartbeat 更新项目心跳配置
func (c *Client) UpdateProjectHeartbeat(ctx context.Context, projectID string, enabled bool, intervalMinutes int, mdContent, agentCode string) (*Project, error) {
	req := UpdateProjectRequest{
		ID:                       projectID,
		HeartbeatEnabled:         &enabled,
		HeartbeatIntervalMinutes: &intervalMinutes,
		HeartbeatMDContent:       &mdContent,
		AgentCode:                &agentCode,
	}
	return c.UpdateProject(ctx, req)
}

// DeleteProject 删除项目
func (c *Client) DeleteProject(ctx context.Context, projectID string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, "/projects?id="+projectID, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return c.handleError(resp)
	}

	return nil
}

// ==================== State Machine APIs ====================
