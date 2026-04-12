package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"input_schema,omitempty"`
}

// MCPServer MCP 服务器响应结构
type MCPServer struct {
	ID            string            `json:"id"`
	Code          string            `json:"code"`
	Name          string            `json:"name"`
	Description   string            `json:"description,omitempty"`
	TransportType string            `json:"transport_type"`
	Command       string            `json:"command,omitempty"`
	Args          []string          `json:"args,omitempty"`
	URL           string            `json:"url,omitempty"`
	EnvVars       map[string]string `json:"env_vars,omitempty"`
	Status        string            `json:"status"`
	Capabilities  []MCPTool         `json:"capabilities,omitempty"`
	LastConnected *int64            `json:"last_connected,omitempty"`
	ErrorMessage  string            `json:"error_message,omitempty"`
	CreatedAt     int64             `json:"created_at"`
	UpdatedAt     int64             `json:"updated_at"`
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
	path := fmt.Sprintf("/mcp/servers?id=%s", id)
	resp, err := c.doRequest(ctx, http.MethodPut, path, req)
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
