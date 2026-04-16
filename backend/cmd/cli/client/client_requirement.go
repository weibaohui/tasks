package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Requirement struct {
	ID                 string                 `json:"id"`
	ProjectID          string                 `json:"project_id"`
	Title              string                 `json:"title"`
	Description        string                 `json:"description"`
	AcceptanceCriteria string                 `json:"acceptance_criteria"`
	TempWorkspaceRoot  string                 `json:"temp_workspace_root"`
	Status             string                 `json:"status"`
	RequirementType    string                 `json:"requirement_type"`
	AssigneeAgentCode  string                 `json:"assignee_agent_code,omitempty"`
	ReplicaAgentCode   string                 `json:"replica_agent_code,omitempty"`
	WorkspacePath      string                 `json:"workspace_path,omitempty"`
	DispatchSessionKey string                 `json:"dispatch_session_key,omitempty"`
	LastError          string                 `json:"last_error,omitempty"`
	CreatedAt          int64                  `json:"created_at"`
	UpdatedAt          int64                  `json:"updated_at"`
	StartedAt          *int64                 `json:"started_at,omitempty"`
	CompletedAt        *int64                 `json:"completed_at,omitempty"`
	ClaudeRuntime      map[string]interface{} `json:"claude_runtime,omitempty"`
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

// ListRequirementsWithParams 获取需求列表（支持更多过滤参数）
func (c *Client) ListRequirementsWithParams(ctx context.Context, params map[string]string) (*ListRequirementsResponse, error) {
	path := "/requirements"
	if len(params) > 0 {
		query := ""
		for k, v := range params {
			if query != "" {
				query += "&"
			}
			query += k + "=" + v
		}
		path += "?" + query
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
	RequirementType    string `json:"requirement_type,omitempty"`
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
	Status             *string `json:"status,omitempty"`
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
