/**
 * UseMCPTool - 加载 MCP Server 工具
 * 用于按需加载 MCP Server 的工具列表
 */
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/llm"
)

// UseMCPTool use_mcp 工具 - 用于按需加载 MCP Server
type UseMCPTool struct {
	mcpService *application.MCPApplicationService
}

// NewUseMCPTool 创建 use_mcp 工具
func NewUseMCPTool(mcpService *application.MCPApplicationService) *UseMCPTool {
	return &UseMCPTool{
		mcpService: mcpService,
	}
}

var _ llm.Tool = (*UseMCPTool)(nil)

// Name 返回工具名称
func (t *UseMCPTool) Name() string {
	return "use_mcp"
}

// Description 返回工具描述
func (t *UseMCPTool) Description() string {
	return `加载并使用指定 MCP Server 的工具。
调用此工具后，该 MCP Server 的所有工具将可在当前对话中使用。
参数 server_code: MCP Server 编码（如 'weather-server', 'file-system'）
参数 action: 操作类型，'load'(加载并返回工具列表) 或 'info'(仅返回 Server 信息)`
}

// Parameters 返回参数 schema
func (t *UseMCPTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"server_code": {
				"type": "string",
				"description": "MCP Server 编码（如 'weather-server', 'file-system'）"
			},
			"action": {
				"type": "string",
				"description": "操作类型：'load'(加载并返回工具列表), 'info'(仅返回 Server 信息，不加载工具)",
				"enum": ["load", "info"]
			}
		},
		"required": ["server_code"]
	}`)
}

// Execute 执行工具
func (t *UseMCPTool) Execute(ctx context.Context, input json.RawMessage) (*llm.ToolResult, error) {
	var args struct {
		ServerCode string `json:"server_code"`
		Action     string `json:"action"`
	}

	if err := json.Unmarshal(input, &args); err != nil {
		return &llm.ToolResult{
			Output: "",
			Error:  fmt.Sprintf("解析参数失败: %v", err),
		}, nil
	}

	if args.ServerCode == "" {
		return &llm.ToolResult{
			Output: "",
			Error:  "server_code 不能为空",
		}, nil
	}

	// 默认操作为 load
	if args.Action == "" {
		args.Action = "load"
	}

	switch args.Action {
	case "load":
		return t.handleLoad(ctx, args.ServerCode)
	case "info":
		return t.handleInfo(ctx, args.ServerCode)
	default:
		return &llm.ToolResult{
			Output: "",
			Error:  fmt.Sprintf("不支持的操作: %s", args.Action),
		}, nil
	}
}

// handleLoad 加载 MCP Server
func (t *UseMCPTool) handleLoad(ctx context.Context, serverCode string) (*llm.ToolResult, error) {
	// 查找 server
	servers, err := t.mcpService.ListServers(ctx)
	if err != nil {
		return &llm.ToolResult{
			Output: "",
			Error:  fmt.Sprintf("获取 MCP Server 列表失败: %v", err),
		}, nil
	}

	var server *domain.MCPServer
	for _, s := range servers {
		if s.Code() == serverCode {
			server = s
			break
		}
	}

	if server == nil {
		return &llm.ToolResult{
			Output: "",
			Error:  fmt.Sprintf("MCP Server '%s' 不存在", serverCode),
		}, nil
	}

	if server.Status() != "active" {
		return &llm.ToolResult{
			Output: "",
			Error:  fmt.Sprintf("MCP Server '%s' 未激活，当前状态: %s", serverCode, server.Status()),
		}, nil
	}

	// 获取工具列表
	tools, err := t.mcpService.ListTools(ctx, server.ID())
	if err != nil {
		return &llm.ToolResult{
			Output: "",
			Error:  fmt.Sprintf("获取工具列表失败: %v", err),
		}, nil
	}

	// 构建响应
	toolInfos := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		toolInfos = append(toolInfos, map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
		})
	}

	message := fmt.Sprintf("MCP Server '%s' 加载成功，包含 %d 个工具。\n可以使用 call_mcp_tool 工具调用这些工具。\n示例：call_mcp_tool(server_code='%s', tool_name='工具名', params={...})",
		server.Name(), len(tools), server.Code())

	result := map[string]interface{}{
		"success":     true,
		"server_code": server.Code(),
		"server_name": server.Name(),
		"message":     message,
		"tools":       toolInfos,
		"tool_count":  len(tools),
		"usage":       "使用 call_mcp_tool(server_code, tool_name, params) 调用工具",
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &llm.ToolResult{
		Output: string(resultJSON),
		Error:  "",
	}, nil
}

// handleInfo 获取 Server 信息（不加载）
func (t *UseMCPTool) handleInfo(ctx context.Context, serverCode string) (*llm.ToolResult, error) {
	servers, err := t.mcpService.ListServers(ctx)
	if err != nil {
		return &llm.ToolResult{
			Output: "",
			Error:  fmt.Sprintf("获取 MCP Server 列表失败: %v", err),
		}, nil
	}

	var server *domain.MCPServer
	for _, s := range servers {
		if s.Code() == serverCode {
			server = s
			break
		}
	}

	if server == nil {
		return &llm.ToolResult{
			Output: "",
			Error:  fmt.Sprintf("MCP Server '%s' 不存在", serverCode),
		}, nil
	}

	result := map[string]interface{}{
		"success":     true,
		"server_code": server.Code(),
		"server_name": server.Name(),
		"description": server.Description(),
		"status":      server.Status(),
		"tool_count":  len(server.Capabilities()),
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &llm.ToolResult{
		Output: string(resultJSON),
		Error:  "",
	}, nil
}
