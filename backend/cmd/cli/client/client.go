// Package client 提供 TaskManager API 的 HTTP 客户端封装
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/weibh/taskmanager/infrastructure/config"
)

// Client TaskManager API 客户端
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// New 创建新的 API 客户端
func New() *Client {
	return NewWithConfig(config.GetAPIBaseURL(), config.GetAPIToken())
}

// NewWithConfig 使用指定配置创建客户端
func NewWithConfig(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// SetTimeout 设置 HTTP 超时
func (c *Client) SetTimeout(timeout time.Duration) {
	c.http.Timeout = timeout
}

// doRequest 执行带认证的 HTTP 请求
// 会自动添加 Bearer Token 到请求头
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	if c.token == "" {
		return nil, fmt.Errorf("API token not configured, please set api.token in ~/.taskmanager/config.yaml")
	}

	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body failed: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonBytes)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	return c.http.Do(req)
}

// doRequestWithoutAuth 执行无需认证的 HTTP 请求
func (c *Client) doRequestWithoutAuth(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body failed: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonBytes)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	return c.http.Do(req)
}

// handleError 处理 HTTP 错误响应
func (c *Client) handleError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("unauthorized: invalid or expired token")
	case http.StatusForbidden:
		return fmt.Errorf("forbidden: insufficient permissions")
	case http.StatusNotFound:
		return fmt.Errorf("not found")
	case http.StatusBadRequest:
		return fmt.Errorf("bad request: %s", string(body))
	default:
		return fmt.Errorf("%s: %s", resp.Status, string(body))
	}
}

// Requirement 需求响应结构
type Requirement struct {
	ID                   string                 `json:"id"`
	ProjectID            string                 `json:"project_id"`
	Title                string                 `json:"title"`
	Description          string                 `json:"description"`
	AcceptanceCriteria   string                 `json:"acceptance_criteria"`
	TempWorkspaceRoot    string                 `json:"temp_workspace_root"`
	Status               string                 `json:"status"`
	RequirementType      string                 `json:"requirement_type"`
	AssigneeAgentCode    string                 `json:"assignee_agent_code,omitempty"`
	ReplicaAgentCode     string                 `json:"replica_agent_code,omitempty"`
	WorkspacePath        string                 `json:"workspace_path,omitempty"`
	DispatchSessionKey   string                 `json:"dispatch_session_key,omitempty"`
	LastError            string                 `json:"last_error,omitempty"`
	CreatedAt            int64                  `json:"created_at"`
	UpdatedAt            int64                  `json:"updated_at"`
	StartedAt            *int64                 `json:"started_at,omitempty"`
	CompletedAt          *int64                 `json:"completed_at,omitempty"`
	ClaudeRuntime        map[string]interface{} `json:"claude_runtime,omitempty"`
}

// ListRequirementsResponse 需求列表响应
type ListRequirementsResponse []Requirement

// ListRequirements 获取需求列表
func (c *Client) ListRequirements(ctx context.Context, projectID string) (*ListRequirementsResponse, error) {
	path := "/requirements"
	if projectID != "" {
		path += "?project_id=" + projectID
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result ListRequirementsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// CreateRequirementRequest 创建需求请求
type CreateRequirementRequest struct {
	ProjectID          string `json:"project_id"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
	TempWorkspaceRoot  string `json:"temp_workspace_root,omitempty"`
}

// CreateRequirement 创建需求
func (c *Client) CreateRequirement(ctx context.Context, req CreateRequirementRequest) (*Requirement, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/requirements", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleError(resp)
	}

	var result Requirement
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// GetRequirement 获取需求详情
func (c *Client) GetRequirement(ctx context.Context, id string) (*Requirement, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/requirements?id="+id, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result Requirement
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// UpdateRequirementRequest 更新需求请求
type UpdateRequirementRequest struct {
	ID                 string  `json:"id"`
	Title              *string `json:"title,omitempty"`
	Description        *string `json:"description,omitempty"`
	AcceptanceCriteria *string `json:"acceptance_criteria,omitempty"`
	TempWorkspaceRoot  *string `json:"temp_workspace_root,omitempty"`
}

// UpdateRequirement 更新需求
func (c *Client) UpdateRequirement(ctx context.Context, req UpdateRequirementRequest) (*Requirement, error) {
	resp, err := c.doRequest(ctx, http.MethodPut, "/requirements", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result Requirement
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// DispatchRequirementRequest 派发需求请求
type DispatchRequirementRequest struct {
	RequirementID string `json:"requirement_id"`
	AgentCode     string `json:"agent_code"`
	ChannelCode   string `json:"channel_code"`
	SessionKey    string `json:"session_key"`
}

// DispatchResult 派发结果
type DispatchResult struct {
	RequirementID string `json:"requirement_id"`
	Status        string `json:"status"`
	AgentCode     string `json:"agent_code,omitempty"`
	SessionKey    string `json:"session_key,omitempty"`
	WorkspacePath string `json:"workspace_path,omitempty"`
	Message       string `json:"message,omitempty"`
}

// DispatchRequirement 派发需求
func (c *Client) DispatchRequirement(ctx context.Context, req DispatchRequirementRequest) (*DispatchResult, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/requirements/dispatch", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result DispatchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// CompleteRequirement 完成需求（报告PR已打开）
func (c *Client) CompleteRequirement(ctx context.Context, requirementID string) (*Requirement, error) {
	reqBody := map[string]string{
		"requirement_id": requirementID,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/requirements/pr", reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result Requirement
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// ResetRequirement 重置需求
func (c *Client) ResetRequirement(ctx context.Context, requirementID string) (*Requirement, error) {
	reqBody := map[string]string{
		"requirement_id": requirementID,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/requirements/reset", reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result Requirement
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// RedispatchRequirement 重新派发需求
func (c *Client) RedispatchRequirement(ctx context.Context, requirementID string) (*Requirement, error) {
	reqBody := map[string]string{
		"requirement_id": requirementID,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/requirements/redispatch", reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result Requirement
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// CopyAndDispatchRequirement 复制并派发需求
func (c *Client) CopyAndDispatchRequirement(ctx context.Context, requirementID string) (*Requirement, error) {
	reqBody := map[string]string{
		"requirement_id": requirementID,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/requirements/copy-and-dispatch", reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result Requirement
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// Task 任务响应结构
type Task struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	ParentID string `json:"parent_id,omitempty"`
	Type     string `json:"type"`
}

// GetRequirementTasks 获取需求的任务列表
func (c *Client) GetRequirementTasks(ctx context.Context, requirementID string) ([]Task, string, error) {
	// 首先获取需求详情以获取 session_key
	req, err := c.GetRequirement(ctx, requirementID)
	if err != nil {
		return nil, "", fmt.Errorf("get requirement failed: %w", err)
	}

	sessionKey := req.DispatchSessionKey
	if sessionKey == "" {
		return []Task{}, sessionKey, nil
	}

	// 获取所有任务（目前没有按 session_key 过滤的 API，先获取所有）
	resp, err := c.doRequest(ctx, http.MethodGet, "/tasks/all", nil)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", c.handleError(resp)
	}

	var allTasks []Task
	if err := json.NewDecoder(resp.Body).Decode(&allTasks); err != nil {
		return nil, "", fmt.Errorf("decode response failed: %w", err)
	}

	// 过滤与需求相关的任务
	var tasks []Task
	for _, task := range allTasks {
		// 这里需要 Task 结构体包含 SessionKey 字段才能正确过滤
		// 目前 API 返回的 Task 可能不包含此字段，需要服务端支持
		_ = task
	}

	return tasks, sessionKey, nil
}

// Agent Agent 响应结构
type Agent struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Code     string `json:"code"`
	AgentType string `json:"agent_type"`
	IsActive bool   `json:"is_active"`
}

// ListAgents 获取 Agent 列表
func (c *Client) ListAgents(ctx context.Context) ([]Agent, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/agents", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result []Agent
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return result, nil
}

// Project Project 响应结构
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
