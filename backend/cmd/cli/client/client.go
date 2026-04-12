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
// StateTrigger 状态触发器指南
// HookConfig Hook配置响应结构
// GetAgent 获取 Agent 详情
// Channel 渠道响应结构
// ProviderModelInfo Provider 模型信息
// MCPTool MCP 工具
// Session 会话响应结构
// User 用户响应结构
// Skill 技能响应结构
