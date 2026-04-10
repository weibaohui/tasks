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

// Project 项目响应结构
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

	if resp.StatusCode != http.StatusOK {
		return c.handleError(resp)
	}

	return nil
}

// ==================== State Machine APIs ====================

// StateTrigger 状态触发器指南
type StateTrigger struct {
	Trigger     string `json:"trigger"`
	Description string `json:"description,omitempty"`
	Condition   string `json:"condition,omitempty"`
}

// State 状态机状态
type State struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	IsFinal         bool           `json:"is_final"`
	AIGuide         string         `json:"ai_guide,omitempty"`
	AutoInit        string         `json:"auto_init,omitempty"`
	SuccessCriteria string         `json:"success_criteria,omitempty"`
	FailureCriteria string         `json:"failure_criteria,omitempty"`
	Triggers        []StateTrigger `json:"triggers,omitempty"`
}

// StateMachineConfig 状态机配置
type StateMachineConfig struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	InitialState string `json:"initial_state"`
	States       []State `json:"states"`
	Transitions []struct {
		From        string `json:"from"`
		To          string `json:"to"`
		Trigger     string `json:"trigger"`
		Description string `json:"description,omitempty"`
	} `json:"transitions"`
}

// StateMachine 状态机响应结构
type StateMachine struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Config      StateMachineConfig `json:"config"`
	CreatedAt   string             `json:"created_at"`
	UpdatedAt   string             `json:"updated_at"`
}

// ListStateMachines 获取状态机列表
func (c *Client) ListStateMachines(ctx context.Context) ([]StateMachine, error) {
	path := "/state-machines"
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result []StateMachine
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return result, nil
}

// CreateStateMachine 创建状态机
func (c *Client) CreateStateMachine(ctx context.Context, name, description, config string) (*StateMachine, error) {
	path := "/state-machines"
	reqBody := map[string]string{
		"name":        name,
		"description": description,
		"config":      config,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleError(resp)
	}

	var result StateMachine
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// DeleteStateMachine 删除状态机
func (c *Client) DeleteStateMachine(ctx context.Context, id string) error {
	path := "/state-machines/" + id
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

// GetStateMachine 获取状态机详情
func (c *Client) GetStateMachine(ctx context.Context, id string) (*StateMachine, error) {
	path := "/state-machines/" + id
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result StateMachine
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// TriggerTransition 触发状态转换
func (c *Client) TriggerTransition(ctx context.Context, requirementID, trigger, triggeredBy, remark string, metadata map[string]interface{}) error {
	path := "/requirements/" + requirementID + "/transitions"
	reqBody := map[string]interface{}{
		"trigger":      trigger,
		"triggered_by": triggeredBy,
		"remark":       remark,
	}
	if metadata != nil {
		reqBody["metadata"] = metadata
	}

	resp, err := c.doRequest(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.handleError(resp)
	}

	return nil
}

// GetRequirementState 获取需求状态
type RequirementState struct {
	ID             string `json:"id"`
	RequirementID  string `json:"requirement_id"`
	StateMachineID string `json:"state_machine_id"`
	CurrentState   string `json:"current_state"`
	CurrentStateName string `json:"current_state_name"`
}

func (c *Client) GetRequirementState(ctx context.Context, requirementID string) (*RequirementState, error) {
	path := "/requirements/" + requirementID + "/state"
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result RequirementState
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// InitializeRequirementState 初始化需求状态
func (c *Client) InitializeRequirementState(ctx context.Context, requirementID, stateMachineID string) (*RequirementState, error) {
	path := "/requirements/" + requirementID + "/state"
	reqBody := map[string]string{
		"state_machine_id": stateMachineID,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleError(resp)
	}

	var result RequirementState
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// GetTransitionHistory 获取转换历史
type TransitionLog struct {
	ID            string `json:"id"`
	RequirementID string `json:"requirement_id"`
	FromState    string `json:"from_state"`
	ToState      string `json:"to_state"`
	Trigger      string `json:"trigger"`
	TriggeredBy  string `json:"triggered_by"`
	Result       string `json:"result"`
	CreatedAt    string `json:"created_at"`
}

func (c *Client) GetTransitionHistory(ctx context.Context, requirementID string) ([]TransitionLog, error) {
	path := "/requirements/" + requirementID + "/transitions/history"
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result []TransitionLog
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return result, nil
}

// GetProjectStateSummary 获取项目状态统计
type StateSummary struct {
	StateID   string `json:"state_id"`
	StateName string `json:"state_name"`
	Count     int    `json:"count"`
}

func (c *Client) GetProjectStateSummary(ctx context.Context, projectID string) ([]StateSummary, error) {
	path := "/projects/" + projectID + "/requirements/states/summary"
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result []StateSummary
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return result, nil
}

// BindType 绑定需求类型到状态机
func (c *Client) BindType(ctx context.Context, stateMachineID, requirementType string) error {
	path := "/state-machines/" + stateMachineID + "/bind"
	reqBody := map[string]string{
		"requirement_type": requirementType,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return c.handleError(resp)
	}

	return nil
}

// UnbindType 解绑需求类型
func (c *Client) UnbindType(ctx context.Context, stateMachineID, requirementType string) error {
	path := "/state-machines/" + stateMachineID + "/bind/" + requirementType
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

// ==================== Hook APIs ====================

// HookConfig Hook配置响应结构
type HookConfig struct {
	ID           string `json:"id"`
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	TriggerPoint string `json:"trigger_point"`
	ActionType  string `json:"action_type"`
	Enabled     bool   `json:"enabled"`
	Priority    int    `json:"priority"`
	ActionConfig string `json:"action_config"`
}

// ListHookConfigs 获取Hook配置列表
func (c *Client) ListHookConfigs(ctx context.Context, projectID string) ([]HookConfig, error) {
	path := "/hook-configs"
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

	var result []HookConfig
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return result, nil
}

// DeleteRequirement 删除需求
func (c *Client) DeleteRequirement(ctx context.Context, requirementID string) error {
	path := "/requirements?id=" + requirementID
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

// ==================== Requirement Type APIs ====================

// RequirementType 需求类型响应结构
type RequirementType struct {
	ID             string `json:"id"`
	ProjectID      string `json:"project_id"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Icon           string `json:"icon"`
	Color          string `json:"color"`
	SortOrder      int    `json:"sort_order"`
	StateMachineID string `json:"state_machine_id"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

// ListRequirementTypes 获取项目下的需求类型列表
func (c *Client) ListRequirementTypes(ctx context.Context, projectID string) ([]RequirementType, error) {
	path := "/requirement-types?project_id=" + projectID
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result []RequirementType
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return result, nil
}

// CreateRequirementTypeRequest 创建需求类型请求
type CreateRequirementTypeRequest struct {
	ProjectID   string `json:"project_id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Color       string `json:"color"`
}

// CreateRequirementType 创建需求类型
func (c *Client) CreateRequirementType(ctx context.Context, req CreateRequirementTypeRequest) (*RequirementType, error) {
	path := "/requirement-types"
	resp, err := c.doRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleError(resp)
	}

	var result RequirementType
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// ==================== Agent APIs ====================

// GetAgent 获取 Agent 详情
func (c *Client) GetAgent(ctx context.Context, id string) (*Agent, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/agents?id="+id, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result Agent
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// CreateAgentAPIRequest 创建 Agent 请求
type CreateAgentAPIRequest struct {
	UserCode            string  `json:"user_code"`
	Name                string  `json:"name"`
	AgentType           string  `json:"agent_type"`
	Description         string  `json:"description"`
	IdentityContent     string  `json:"identity_content"`
	SoulContent         string  `json:"soul_content"`
	AgentsContent       string  `json:"agents_content"`
	UserContent         string  `json:"user_content"`
	ToolsContent        string  `json:"tools_content"`
	Model               string  `json:"model"`
	LLMProviderID       string  `json:"llm_provider_id"`
	MaxTokens           int     `json:"max_tokens"`
	Temperature         float64 `json:"temperature"`
	MaxIterations       int     `json:"max_iterations"`
	HistoryMessages     int     `json:"history_messages"`
	SkillsList          []string `json:"skills_list"`
	ToolsList           []string `json:"tools_list"`
	IsDefault           bool    `json:"is_default"`
	EnableThinkingProcess bool   `json:"enable_thinking_process"`
}

// CreateAgent 创建 Agent
func (c *Client) CreateAgent(ctx context.Context, req CreateAgentAPIRequest) (*Agent, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/agents", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleError(resp)
	}

	var result Agent
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// UpdateAgentAPIRequest 更新 Agent 请求
type UpdateAgentAPIRequest struct {
	Name                  *string   `json:"name,omitempty"`
	AgentType             *string   `json:"agent_type,omitempty"`
	Description           *string   `json:"description,omitempty"`
	IdentityContent       *string   `json:"identity_content,omitempty"`
	SoulContent           *string   `json:"soul_content,omitempty"`
	AgentsContent         *string   `json:"agents_content,omitempty"`
	UserContent           *string   `json:"user_content,omitempty"`
	ToolsContent          *string   `json:"tools_content,omitempty"`
	Model                 *string   `json:"model,omitempty"`
	LLMProviderID         *string   `json:"llm_provider_id,omitempty"`
	MaxTokens             *int      `json:"max_tokens,omitempty"`
	Temperature           *float64  `json:"temperature,omitempty"`
	MaxIterations         *int      `json:"max_iterations,omitempty"`
	HistoryMessages       *int      `json:"history_messages,omitempty"`
	SkillsList            []string  `json:"skills_list,omitempty"`
	ToolsList             []string  `json:"tools_list,omitempty"`
	IsActive              *bool     `json:"is_active,omitempty"`
	IsDefault             *bool     `json:"is_default,omitempty"`
	EnableThinkingProcess *bool     `json:"enable_thinking_process,omitempty"`
}

// UpdateAgent 更新 Agent
func (c *Client) UpdateAgent(ctx context.Context, id string, req UpdateAgentAPIRequest) (*Agent, error) {
	resp, err := c.doRequest(ctx, http.MethodPut, "/agents", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result Agent
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// DeleteAgent 删除 Agent
func (c *Client) DeleteAgent(ctx context.Context, id string) error {
	path := "/agents?id=" + id
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

// ==================== Channel APIs ====================

// Channel 渠道响应结构
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
	resp, err := c.doRequest(ctx, http.MethodPut, "/channels", req)
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

// ProviderModelInfo Provider 模型信息
type ProviderModelInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MaxTokens int    `json:"max_tokens"`
}

// LLMProvider LLM Provider 响应结构
type LLMProvider struct {
	ID              string               `json:"id"`
	UserCode        string               `json:"user_code"`
	ProviderKey     string               `json:"provider_key"`
	ProviderName    string               `json:"provider_name"`
	APIBase         string               `json:"api_base"`
	ProviderType    string               `json:"provider_type"`
	ExtraHeaders    map[string]string    `json:"extra_headers"`
	SupportedModels []ProviderModelInfo   `json:"supported_models"`
	DefaultModel    string               `json:"default_model"`
	IsDefault       bool                 `json:"is_default"`
	Priority        int                  `json:"priority"`
	AutoMerge       bool                 `json:"auto_merge"`
	IsActive        bool                 `json:"is_active"`
	CreatedAt       int64                `json:"created_at"`
	UpdatedAt       int64                `json:"updated_at"`
}

// ListProviders 获取 Provider 列表
func (c *Client) ListProviders(ctx context.Context, userCode string) ([]LLMProvider, error) {
	path := "/providers"
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

	var result []LLMProvider
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return result, nil
}

// CreateProviderAPIRequest 创建 Provider 请求
type CreateProviderAPIRequest struct {
	UserCode        string              `json:"user_code"`
	ProviderKey     string              `json:"provider_key"`
	ProviderName    string              `json:"provider_name"`
	APIKey          string              `json:"api_key"`
	APIBase         string              `json:"api_base"`
	ProviderType    string              `json:"provider_type"`
	ExtraHeaders    map[string]string   `json:"extra_headers"`
	SupportedModels []ProviderModelInfo  `json:"supported_models"`
	DefaultModel    string              `json:"default_model"`
	IsDefault       bool                `json:"is_default"`
	Priority        int                 `json:"priority"`
	AutoMerge       bool                `json:"auto_merge"`
}

// CreateProvider 创建 Provider
func (c *Client) CreateProvider(ctx context.Context, req CreateProviderAPIRequest) (*LLMProvider, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/providers", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleError(resp)
	}

	var result LLMProvider
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// UpdateProviderAPIRequest 更新 Provider 请求
type UpdateProviderAPIRequest struct {
	ProviderKey     *string             `json:"provider_key,omitempty"`
	ProviderName    *string             `json:"provider_name,omitempty"`
	APIKey          *string             `json:"api_key,omitempty"`
	APIBase         *string             `json:"api_base,omitempty"`
	ProviderType    *string             `json:"provider_type,omitempty"`
	ExtraHeaders    *map[string]string  `json:"extra_headers,omitempty"`
	SupportedModels *[]ProviderModelInfo `json:"supported_models,omitempty"`
	DefaultModel    *string              `json:"default_model,omitempty"`
	IsDefault       *bool                `json:"is_default,omitempty"`
	Priority        *int                `json:"priority,omitempty"`
	AutoMerge       *bool               `json:"auto_merge,omitempty"`
	IsActive        *bool               `json:"is_active,omitempty"`
}

// UpdateProvider 更新 Provider
func (c *Client) UpdateProvider(ctx context.Context, id string, req UpdateProviderAPIRequest) (*LLMProvider, error) {
	resp, err := c.doRequest(ctx, http.MethodPut, "/providers", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result LLMProvider
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// DeleteProvider 删除 Provider
func (c *Client) DeleteProvider(ctx context.Context, id string) error {
	path := "/providers?id=" + id
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

// TestProviderResult 测试 Provider 连接结果
type TestProviderResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// TestProvider 测试 Provider 连接
func (c *Client) TestProvider(ctx context.Context, id string) (*TestProviderResult, error) {
	path := "/providers/test?id=" + id
	resp, err := c.doRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result TestProviderResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// ==================== MCP Server APIs ====================

// MCPTool MCP 工具
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"input_schema,omitempty"`
}

// MCPServer MCP 服务器响应结构
type MCPServer struct {
	ID             string                 `json:"id"`
	Code           string                 `json:"code"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	TransportType  string                 `json:"transport_type"`
	Command        string                 `json:"command,omitempty"`
	Args           []string               `json:"args,omitempty"`
	URL            string                 `json:"url,omitempty"`
	EnvVars        map[string]string      `json:"env_vars,omitempty"`
	Status         string                 `json:"status"`
	Capabilities   []MCPTool              `json:"capabilities,omitempty"`
	LastConnected  *int64                  `json:"last_connected,omitempty"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	CreatedAt      int64                  `json:"created_at"`
	UpdatedAt      int64                  `json:"updated_at"`
}

// ListMCPServers 获取 MCP 服务器列表
func (c *Client) ListMCPServers(ctx context.Context) ([]MCPServer, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/mcp/servers", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result []MCPServer
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return result, nil
}

// CreateMCPServerAPIRequest 创建 MCP 服务器请求
type CreateMCPServerAPIRequest struct {
	Code          string            `json:"code"`
	Name          string            `json:"name"`
	Description   string            `json:"description,omitempty"`
	TransportType string            `json:"transport_type"`
	Command       string            `json:"command,omitempty"`
	Args          []string          `json:"args,omitempty"`
	URL           string            `json:"url,omitempty"`
	EnvVars       map[string]string `json:"env_vars,omitempty"`
}

// CreateMCPServer 创建 MCP 服务器
func (c *Client) CreateMCPServer(ctx context.Context, req CreateMCPServerAPIRequest) (*MCPServer, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/mcp/servers", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleError(resp)
	}

	var result MCPServer
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// UpdateMCPServerAPIRequest 更新 MCP 服务器请求
type UpdateMCPServerAPIRequest struct {
	Name          *string            `json:"name,omitempty"`
	Description   *string            `json:"description,omitempty"`
	TransportType *string            `json:"transport_type,omitempty"`
	Command       *string            `json:"command,omitempty"`
	Args          *[]string          `json:"args,omitempty"`
	URL           *string            `json:"url,omitempty"`
	EnvVars       *map[string]string `json:"env_vars,omitempty"`
}

// UpdateMCPServer 更新 MCP 服务器
func (c *Client) UpdateMCPServer(ctx context.Context, id string, req UpdateMCPServerAPIRequest) (*MCPServer, error) {
	resp, err := c.doRequest(ctx, http.MethodPut, "/mcp/servers", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result MCPServer
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// DeleteMCPServer 删除 MCP 服务器
func (c *Client) DeleteMCPServer(ctx context.Context, id string) error {
	path := "/mcp/servers?id=" + id
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

// TestMCPServer 测试 MCP 服务器连接
func (c *Client) TestMCPServer(ctx context.Context, id string) error {
	path := "/mcp/servers/test?id=" + id
	resp, err := c.doRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.handleError(resp)
	}

	return nil
}

// RefreshMCPServer 刷新 MCP 服务器工具能力
func (c *Client) RefreshMCPServer(ctx context.Context, id string) error {
	path := "/mcp/servers/refresh?id=" + id
	resp, err := c.doRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.handleError(resp)
	}

	return nil
}

// ==================== Session APIs ====================

// Session 会话响应结构
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

// User 用户响应结构
type User struct {
	ID         string `json:"id"`
	UserCode   string `json:"user_code"`
	Username   string `json:"username"`
	Password   string `json:"password,omitempty"`
	IsAdmin    bool   `json:"is_admin"`
	IsActive   bool   `json:"is_active"`
	Email      string `json:"email,omitempty"`
	Department string `json:"department,omitempty"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

// ListUsers 获取用户列表
func (c *Client) ListUsers(ctx context.Context) ([]User, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/users", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result []User
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return result, nil
}

// CreateUserAPIRequest 创建用户请求
type CreateUserAPIRequest struct {
	UserCode   string `json:"user_code"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	IsAdmin    bool   `json:"is_admin"`
	IsActive   bool   `json:"is_active"`
	Email      string `json:"email,omitempty"`
	Department string `json:"department,omitempty"`
}

// CreateUser 创建用户
func (c *Client) CreateUser(ctx context.Context, req CreateUserAPIRequest) (*User, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/users", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleError(resp)
	}

	var result User
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// UpdateUserAPIRequest 更新用户请求
type UpdateUserAPIRequest struct {
	Username   *string `json:"username,omitempty"`
	Password   *string `json:"password,omitempty"`
	IsAdmin    *bool   `json:"is_admin,omitempty"`
	IsActive   *bool   `json:"is_active,omitempty"`
	Email      *string `json:"email,omitempty"`
	Department *string `json:"department,omitempty"`
}

// UpdateUser 更新用户
func (c *Client) UpdateUser(ctx context.Context, id string, req UpdateUserAPIRequest) (*User, error) {
	resp, err := c.doRequest(ctx, http.MethodPut, "/users", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result User
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}

// DeleteUser 删除用户
func (c *Client) DeleteUser(ctx context.Context, id string) error {
	path := "/users?id=" + id
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

// ==================== Skill APIs ====================

// Skill 技能响应结构
type Skill struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
}

// ListSkills 获取技能列表
func (c *Client) ListSkills(ctx context.Context) ([]Skill, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/skills", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result struct {
		Items []Skill `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return result.Items, nil
}

// GetSkill 获取技能详情
func (c *Client) GetSkill(ctx context.Context, name string) (*Skill, error) {
	path := "/skills/detail?name=" + name
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	var result Skill
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &result, nil
}
