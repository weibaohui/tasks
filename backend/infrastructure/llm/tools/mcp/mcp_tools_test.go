/**
 * MCP Tools 单元测试
 */
package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/weibh/taskmanager/infrastructure/llm"
)

// mockMCPService - 用于测试的 MCP 服务模拟
type mockMCPService struct {
	listServersFn  func(ctx context.Context) ([]*mockServer, error)
	listToolsFn    func(ctx context.Context, id string) ([]*mockTool, error)
	executeToolFn  func(ctx context.Context, id, name string, params map[string]interface{}) (string, error)
}

type mockServer struct {
	id          string
	code        string
	name        string
	description string
	status      string
	capabilities []*mockTool
}

type mockTool struct {
	name        string
	description string
	inputSchema map[string]interface{}
}

func (m *mockMCPService) listServers(ctx context.Context) ([]*mockServer, error) {
	if m.listServersFn != nil {
		return m.listServersFn(ctx)
	}
	return nil, nil
}

func (m *mockMCPService) listTools(ctx context.Context, id string) ([]*mockTool, error) {
	if m.listToolsFn != nil {
		return m.listToolsFn(ctx, id)
	}
	return nil, nil
}

func (m *mockMCPService) executeTool(ctx context.Context, id, name string, params map[string]interface{}) (string, error) {
	if m.executeToolFn != nil {
		return m.executeToolFn(ctx, id, name, params)
	}
	return "", nil
}

// UseMCPTool tests

func TestUseMCPTool_Name(t *testing.T) {
	tool := NewUseMCPTool(nil)
	if tool.Name() != "use_mcp" {
		t.Errorf("期望名称为 use_mcp, 实际为 %s", tool.Name())
	}
}

func TestUseMCPTool_Description(t *testing.T) {
	tool := NewUseMCPTool(nil)
	desc := tool.Description()
	if desc == "" {
		t.Error("期望有描述")
	}
}

func TestUseMCPTool_Parameters(t *testing.T) {
	tool := NewUseMCPTool(nil)
	params := tool.Parameters()
	if params == nil {
		t.Fatal("期望有参数")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(params, &parsed); err != nil {
		t.Fatalf("参数应该是有效的 JSON: %v", err)
	}

	if parsed["type"] != "object" {
		t.Errorf("期望 type 为 object, 实际为 %v", parsed["type"])
	}

	props, ok := parsed["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("期望有 properties")
	}

	if _, ok := props["server_code"]; !ok {
		t.Error("期望有 server_code 参数")
	}

	if _, ok := props["action"]; !ok {
		t.Error("期望有 action 参数")
	}
}

func TestUseMCPTool_ImplementsToolInterface(t *testing.T) {
	tool := NewUseMCPTool(nil)
	var _ llm.Tool = tool
}

// CallMCPTool tests

func TestCallMCPTool_Name(t *testing.T) {
	tool := NewCallMCPTool(nil)
	if tool.Name() != "call_mcp_tool" {
		t.Errorf("期望名称为 call_mcp_tool, 实际为 %s", tool.Name())
	}
}

func TestCallMCPTool_Description(t *testing.T) {
	tool := NewCallMCPTool(nil)
	desc := tool.Description()
	if desc == "" {
		t.Error("期望有描述")
	}
}

func TestCallMCPTool_Parameters(t *testing.T) {
	tool := NewCallMCPTool(nil)
	params := tool.Parameters()
	if params == nil {
		t.Fatal("期望有参数")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(params, &parsed); err != nil {
		t.Fatalf("参数应该是有效的 JSON: %v", err)
	}

	if parsed["type"] != "object" {
		t.Errorf("期望 type 为 object, 实际为 %v", parsed["type"])
	}

	props, ok := parsed["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("期望有 properties")
	}

	if _, ok := props["server_code"]; !ok {
		t.Error("期望有 server_code 参数")
	}

	if _, ok := props["tool_name"]; !ok {
		t.Error("期望有 tool_name 参数")
	}

	if _, ok := props["params"]; !ok {
		t.Error("期望有 params 参数")
	}
}

func TestCallMCPTool_ImplementsToolInterface(t *testing.T) {
	tool := NewCallMCPTool(nil)
	var _ llm.Tool = tool
}

// Test validation logic

func TestUseMCPTool_Execute_EmptyServerCode(t *testing.T) {
	tool := NewUseMCPTool(nil)

	result, err := tool.Execute(context.Background(), []byte(`{"server_code": ""}`))
	if err != nil {
		t.Fatalf("Execute 不应返回 error: %v", err)
	}

	if result.Error == "" {
		t.Error("期望有错误信息")
	}

	if result.Output != "" {
		t.Error("期望空输出")
	}
}

func TestUseMCPTool_Execute_InvalidJSON(t *testing.T) {
	tool := NewUseMCPTool(nil)

	result, err := tool.Execute(context.Background(), []byte(`invalid json`))
	if err != nil {
		t.Fatalf("Execute 不应返回 error: %v", err)
	}

	if result.Error == "" {
		t.Error("期望有错误信息")
	}
}

func TestUseMCPTool_Execute_UnsupportedAction(t *testing.T) {
	tool := NewUseMCPTool(nil)

	result, err := tool.Execute(context.Background(), []byte(`{"server_code": "test", "action": "unsupported"}`))
	if err != nil {
		t.Fatalf("Execute 不应返回 error: %v", err)
	}

	if result.Error == "" {
		t.Error("期望有错误信息")
	}
}

func TestCallMCPTool_Execute_EmptyServerCode(t *testing.T) {
	tool := NewCallMCPTool(nil)

	result, err := tool.Execute(context.Background(), []byte(`{"server_code": "", "tool_name": "test"}`))
	if err != nil {
		t.Fatalf("Execute 不应返回 error: %v", err)
	}

	if result.Error == "" {
		t.Error("期望有错误信息")
	}
}

func TestCallMCPTool_Execute_EmptyToolName(t *testing.T) {
	tool := NewCallMCPTool(nil)

	result, err := tool.Execute(context.Background(), []byte(`{"server_code": "test", "tool_name": ""}`))
	if err != nil {
		t.Fatalf("Execute 不应返回 error: %v", err)
	}

	if result.Error == "" {
		t.Error("期望有错误信息")
	}
}

func TestCallMCPTool_Execute_InvalidJSON(t *testing.T) {
	tool := NewCallMCPTool(nil)

	result, err := tool.Execute(context.Background(), []byte(`invalid json`))
	if err != nil {
		t.Fatalf("Execute 不应返回 error: %v", err)
	}

	if result.Error == "" {
		t.Error("期望有错误信息")
	}
}
