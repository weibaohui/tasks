package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type ProviderModelInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MaxTokens int    `json:"max_tokens"`
}

// LLMProvider LLM Provider 响应结构
type LLMProvider struct {
	ID              string              `json:"id"`
	UserCode        string              `json:"user_code"`
	ProviderKey     string              `json:"provider_key"`
	ProviderName    string              `json:"provider_name"`
	APIBase         string              `json:"api_base"`
	ProviderType    string              `json:"provider_type"`
	ExtraHeaders    map[string]string   `json:"extra_headers"`
	SupportedModels []ProviderModelInfo `json:"supported_models"`
	DefaultModel    string              `json:"default_model"`
	IsDefault       bool                `json:"is_default"`
	Priority        int                 `json:"priority"`
	AutoMerge       bool                `json:"auto_merge"`
	IsActive        bool                `json:"is_active"`
	CreatedAt       int64               `json:"created_at"`
	UpdatedAt       int64               `json:"updated_at"`
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
	SupportedModels []ProviderModelInfo `json:"supported_models"`
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
	ProviderKey     *string              `json:"provider_key,omitempty"`
	ProviderName    *string              `json:"provider_name,omitempty"`
	APIKey          *string              `json:"api_key,omitempty"`
	APIBase         *string              `json:"api_base,omitempty"`
	ProviderType    *string              `json:"provider_type,omitempty"`
	ExtraHeaders    *map[string]string   `json:"extra_headers,omitempty"`
	SupportedModels *[]ProviderModelInfo `json:"supported_models,omitempty"`
	DefaultModel    *string              `json:"default_model,omitempty"`
	IsDefault       *bool                `json:"is_default,omitempty"`
	Priority        *int                 `json:"priority,omitempty"`
	AutoMerge       *bool                `json:"auto_merge,omitempty"`
	IsActive        *bool                `json:"is_active,omitempty"`
}

// UpdateProvider 更新 Provider
func (c *Client) UpdateProvider(ctx context.Context, id string, req UpdateProviderAPIRequest) (*LLMProvider, error) {
	path := fmt.Sprintf("/providers?id=%s", id)
	resp, err := c.doRequest(ctx, http.MethodPut, path, req)
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
