package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

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
	Name         string  `json:"name"`
	Description  string  `json:"description,omitempty"`
	InitialState string  `json:"initial_state"`
	States       []State `json:"states"`
	Transitions  []struct {
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
	ID               string `json:"id"`
	RequirementID    string `json:"requirement_id"`
	StateMachineID   string `json:"state_machine_id"`
	CurrentState     string `json:"current_state"`
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
	FromState     string `json:"from_state"`
	ToState       string `json:"to_state"`
	Trigger       string `json:"trigger"`
	TriggeredBy   string `json:"triggered_by"`
	Result        string `json:"result"`
	CreatedAt     string `json:"created_at"`
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
