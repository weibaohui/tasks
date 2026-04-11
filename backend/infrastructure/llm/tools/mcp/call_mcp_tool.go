/**
 * CallMCPTool - 调用 MCP 工具
 * 用于调用已加载的 MCP Server 中的具体工具
 */
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/llm"
)

// CallMCPTool 通用 MCP 工具调用器
type CallMCPTool struct {
	mcpService domain.MCPToolService
}

// NewCallMCPTool 创建通用 MCP 工具调用器
func NewCallMCPTool(mcpService domain.MCPToolService) *CallMCPTool {
	return &CallMCPTool{
		mcpService: mcpService,
	}
}

var _ llm.Tool = (*CallMCPTool)(nil)

// Name 返回工具名称
func (t *CallMCPTool) Name() string {
	return "call_mcp_tool"
}

// Description 返回工具描述
func (t *CallMCPTool) Description() string {
	return `调用 MCP Server 中的工具。
参数 server_code: MCP Server 编码（如 'weather-server'）
参数 tool_name: 工具名称（如 'get_current_weather'）
参数 params: 工具参数（JSON 对象，根据具体工具的要求）
示例：call_mcp_tool(server_code='weather-server', tool_name='get_current_weather', params={"city":"北京"})`
}

// Parameters 返回参数 schema
func (t *CallMCPTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"server_code": {
				"type": "string",
				"description": "MCP Server 编码（如 'weather-server'）"
			},
			"tool_name": {
				"type": "string",
				"description": "工具名称（如 'get_current_weather'）"
			},
			"params": {
				"type": "object",
				"description": "工具参数（JSON 对象，根据具体工具的要求）"
			}
		},
		"required": ["server_code", "tool_name"]
	}`)
}

// Execute 执行工具
func (t *CallMCPTool) Execute(ctx context.Context, input json.RawMessage) (*llm.ToolResult, error) {
	var args struct {
		ServerCode string                 `json:"server_code"`
		ToolName   string                 `json:"tool_name"`
		Params     map[string]interface{} `json:"params"`
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

	if args.ToolName == "" {
		return &llm.ToolResult{
			Output: "",
			Error:  "tool_name 不能为空",
		}, nil
	}

	// 规范化 nil params 为空对象
	if args.Params == nil {
		args.Params = map[string]interface{}{}
	}

	// 查找 server
	servers, err := t.mcpService.ListServers(ctx)
	if err != nil {
		return &llm.ToolResult{
			Output: "",
			Error:  fmt.Sprintf("获取 MCP Server 列表失败: %v", err),
		}, nil
	}

	var serverID *domain.MCPServerID
	for _, s := range servers {
		if s.Code() == args.ServerCode {
			id := s.ID()
			serverID = &id
			break
		}
	}

	if serverID == nil {
		return &llm.ToolResult{
			Output: "",
			Error:  fmt.Sprintf("MCP Server '%s' 不存在", args.ServerCode),
		}, nil
	}

	// 执行 MCP 工具
	result, err := t.mcpService.ExecuteTool(ctx, *serverID, args.ToolName, args.Params)
	if err != nil {
		return &llm.ToolResult{
			Output: "",
			Error:  fmt.Sprintf("执行 MCP 工具失败: %v", err),
		}, nil
	}

	return &llm.ToolResult{
		Output: result,
		Error:  "",
	}, nil
}
