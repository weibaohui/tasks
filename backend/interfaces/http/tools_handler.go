package http

import (
	"encoding/json"
	"net/http"
)

// BuiltInTool 内置工具信息
type BuiltInTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ToolsHandler 工具处理器
type ToolsHandler struct{}

// NewToolsHandler 创建工具处理器
func NewToolsHandler() *ToolsHandler {
	return &ToolsHandler{}
}

// ListBuiltInTools 返回内置工具列表
func (h *ToolsHandler) ListBuiltInTools(w http.ResponseWriter, r *http.Request) {
	tools := []BuiltInTool{
		{
			Name:        "use_mcp",
			Description: "加载并使用指定 MCP Server 的工具。参数 server_code: MCP Server 编码",
		},
		{
			Name:        "call_mcp_tool",
			Description: "调用 MCP Server 中的工具。参数 server_code, tool_name, params",
		},
		{
			Name:        "bash",
			Description: "执行 Bash 命令并返回输出结果。参数 command: 要执行的命令, timeout: 超时时间(秒)",
		},
		{
			Name:        "use_skill",
			Description: "加载并使用指定技能。参数 skill_name: 技能名称, action: 操作类型（load/info）",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tools)
}
