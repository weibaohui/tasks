/**
 * LLM Tool 接口定义
 * 支持 LLM 调用外部工具
 */
package llm

import (
	"context"
	"encoding/json"
)

// ToolCall 工具调用请求
type ToolCall struct {
	ID    string          `json:"id"`    // 工具调用 ID
	Name  string          `json:"name"`  // 工具名称
	Input json.RawMessage `json:"input"` // 工具输入参数
}

// ToolResult 工具执行结果
type ToolResult struct {
	ID     string `json:"id"`              // 对应的工具调用 ID
	Output string `json:"output"`          // 工具输出
	Error  string `json:"error,omitempty"` // 错误信息
}

// Tool 工具接口
type Tool interface {
	// Name 返回工具名称
	Name() string
	// Description 返回工具描述，用于 LLM 理解工具用途
	Description() string
	// Parameters 返回工具参数 schema (JSON Schema)
	Parameters() json.RawMessage
	// Execute 执行工具
	Execute(ctx context.Context, input json.RawMessage) (*ToolResult, error)
}

// ToolRegistry 工具注册表
type ToolRegistry struct {
	tools map[string]Tool
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register 注册工具
func (r *ToolRegistry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

// Get 获取工具
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// List 返回所有工具
func (r *ToolRegistry) List() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ToolInfos 返回工具信息列表（用于传递给 LLM）
type ToolInfo struct {
	Type     string      `json:"type"`
	Function FunctionDef `json:"function"`
}

// FunctionDef 函数定义
type FunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// GetToolInfos 返回所有工具的信息
func (r *ToolRegistry) GetToolInfos() []ToolInfo {
	tools := r.List()
	infos := make([]ToolInfo, 0, len(tools))
	for _, tool := range tools {
		infos = append(infos, ToolInfo{
			Type: "function",
			Function: FunctionDef{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  tool.Parameters(),
			},
		})
	}
	return infos
}
