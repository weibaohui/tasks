package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type HookConfig struct {
	ID           string `json:"id"`
	ProjectID    string `json:"project_id"`
	Name         string `json:"name"`
	TriggerPoint string `json:"trigger_point"`
	ActionType   string `json:"action_type"`
	Enabled      bool   `json:"enabled"`
	Priority     int    `json:"priority"`
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
