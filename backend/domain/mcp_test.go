/**
 * MCP 领域模型单元测试
 */
package domain

import (
	"testing"
)

func TestNewMCPServer(t *testing.T) {
	server, err := NewMCPServer(NewMCPServerID("srv-001"), "github", "GitHub MCP Server", MCPTransportSTDIO)

	if err != nil {
		t.Fatalf("创建 MCPServer 失败: %v", err)
	}

	if server.ID() != NewMCPServerID("srv-001") {
		t.Errorf("期望 ID 为 srv-001, 实际为 %s", server.ID())
	}

	if server.Code() != "github" {
		t.Errorf("期望 Code 为 github, 实际为 %s", server.Code())
	}

	if server.Name() != "GitHub MCP Server" {
		t.Errorf("期望 Name 为 GitHub MCP Server, 实际为 %s", server.Name())
	}

	if server.Status() != "inactive" {
		t.Errorf("期望初始状态为 inactive, 实际为 %s", server.Status())
	}

	if server.TransportType() != MCPTransportSTDIO {
		t.Errorf("期望 TransportType 为 stdio, 实际为 %s", server.TransportType())
	}
}

func TestNewMCPServer_EmptyID(t *testing.T) {
	_, err := NewMCPServer(NewMCPServerID(""), "github", "GitHub MCP Server", MCPTransportSTDIO)

	if err != ErrMCPServerIDRequired {
		t.Errorf("期望返回 ErrMCPServerIDRequired, 实际返回 %v", err)
	}
}

func TestNewMCPServer_EmptyCode(t *testing.T) {
	_, err := NewMCPServer(NewMCPServerID("srv-001"), "", "GitHub MCP Server", MCPTransportSTDIO)

	if err != ErrMCPServerCodeRequired {
		t.Errorf("期望返回 ErrMCPServerCodeRequired, 实际返回 %v", err)
	}
}

func TestNewMCPServer_EmptyName(t *testing.T) {
	_, err := NewMCPServer(NewMCPServerID("srv-001"), "github", "", MCPTransportSTDIO)

	if err != ErrMCPServerNameRequired {
		t.Errorf("期望返回 ErrMCPServerNameRequired, 实际返回 %v", err)
	}
}

func TestMCPServer_UpdateProfile(t *testing.T) {
	server, _ := NewMCPServer(NewMCPServerID("srv-001"), "github", "GitHub", MCPTransportSTDIO)

	server.UpdateProfile("GitHub MCP", "A MCP server for GitHub", MCPTransportHTTP, "npx", "http://localhost:8080", []string{"arg1"}, map[string]string{"KEY": "value"})

	if server.Name() != "GitHub MCP" {
		t.Errorf("期望 Name 为 GitHub MCP, 实际为 %s", server.Name())
	}

	if server.Description() != "A MCP server for GitHub" {
		t.Errorf("期望 Description 为 A MCP server for GitHub, 实际为 %s", server.Description())
	}

	if server.TransportType() != MCPTransportHTTP {
		t.Errorf("期望 TransportType 为 http, 实际为 %s", server.TransportType())
	}

	if server.Command() != "npx" {
		t.Errorf("期望 Command 为 npx, 实际为 %s", server.Command())
	}

	if server.URL() != "http://localhost:8080" {
		t.Errorf("期望 URL 为 http://localhost:8080, 实际为 %s", server.URL())
	}

	if len(server.Args()) != 1 || server.Args()[0] != "arg1" {
		t.Errorf("期望 Args 为 [arg1], 实际为 %v", server.Args())
	}

	if server.EnvVars()["KEY"] != "value" {
		t.Errorf("期望 EnvVars[KEY] 为 value, 实际为 %s", server.EnvVars()["KEY"])
	}
}

func TestMCPServer_UpdateProfile_Partial(t *testing.T) {
	server, _ := NewMCPServer(NewMCPServerID("srv-001"), "github", "GitHub", MCPTransportSTDIO)
	server.UpdateProfile("NewName", "", "", "", "", nil, nil)

	if server.Name() != "NewName" {
		t.Errorf("期望 Name 为 NewName, 实际为 %s", server.Name())
	}

	// 空字符串会更新
	if server.Description() != "" {
		t.Errorf("期望 Description 为空, 实际为 %s", server.Description())
	}

	// 空传输类型不更新
	if server.TransportType() != MCPTransportSTDIO {
		t.Errorf("期望 TransportType 保持为 stdio, 实际为 %s", server.TransportType())
	}
}

func TestMCPServer_SetStatus(t *testing.T) {
	server, _ := NewMCPServer(NewMCPServerID("srv-001"), "github", "GitHub", MCPTransportSTDIO)

	server.SetStatus("active", "")

	if server.Status() != "active" {
		t.Errorf("期望 Status 为 active, 实际为 %s", server.Status())
	}

	if server.LastConnectedAt() == nil {
		t.Error("期望 LastConnectedAt 不为空")
	}
}

func TestMCPServer_SetStatus_Error(t *testing.T) {
	server, _ := NewMCPServer(NewMCPServerID("srv-001"), "github", "GitHub", MCPTransportSTDIO)

	server.SetStatus("error", "connection refused")

	if server.Status() != "error" {
		t.Errorf("期望 Status 为 error, 实际为 %s", server.Status())
	}

	if server.ErrorMessage() != "connection refused" {
		t.Errorf("期望 ErrorMessage 为 connection refused, 实际为 %s", server.ErrorMessage())
	}

	if server.LastConnectedAt() != nil {
		t.Error("error 状态不应设置 LastConnectedAt")
	}
}

func TestMCPServer_SetCapabilities(t *testing.T) {
	server, _ := NewMCPServer(NewMCPServerID("srv-001"), "github", "GitHub", MCPTransportSTDIO)

	tools := []MCPTool{
		{Name: "list_repos", Description: "List repositories"},
		{Name: "create_issue", Description: "Create an issue"},
	}

	server.SetCapabilities(tools)

	caps := server.Capabilities()
	if len(caps) != 2 {
		t.Errorf("期望 2 个 capabilities, 实际为 %d", len(caps))
	}

	if caps[0].Name != "list_repos" {
		t.Errorf("期望第一个 tool 为 list_repos, 实际为 %s", caps[0].Name)
	}
}

func TestMCPServer_Capabilities_ReturnsCopy(t *testing.T) {
	server, _ := NewMCPServer(NewMCPServerID("srv-001"), "github", "GitHub", MCPTransportSTDIO)

	tools := []MCPTool{{Name: "tool1", Description: "Tool 1"}}
	server.SetCapabilities(tools)

	caps1 := server.Capabilities()
	caps1[0].Name = "modified"

	caps2 := server.Capabilities()
	if caps2[0].Name == "modified" {
		t.Error("Capabilities 应返回拷贝，不应受外部修改影响")
	}
}

func TestMCPServer_ToSnapshot(t *testing.T) {
	server, _ := NewMCPServer(NewMCPServerID("srv-001"), "github", "GitHub", MCPTransportSTDIO)
	server.UpdateProfile("GitHub MCP", "desc", MCPTransportHTTP, "cmd", "url", []string{"a"}, map[string]string{"K": "v"})
	server.SetStatus("active", "")

	snap := server.ToSnapshot()

	if snap.ID != server.ID() {
		t.Errorf("ID 不匹配")
	}
	if snap.Code != "github" {
		t.Errorf("Code 不匹配")
	}
	if snap.Name != "GitHub MCP" {
		t.Errorf("Name 不匹配")
	}
	if snap.Status != "active" {
		t.Errorf("Status 不匹配")
	}
}

func TestMCPServer_FromSnapshot(t *testing.T) {
	server, _ := NewMCPServer(NewMCPServerID("srv-001"), "github", "GitHub", MCPTransportSTDIO)

	snap := MCPServerSnapshot{
		ID:            NewMCPServerID("srv-002"),
		Code:          "gitlab",
		Name:          "GitLab MCP",
		Description:   "GitLab MCP Server",
		TransportType: MCPTransportHTTP,
		Command:       "npx",
		URL:           "http://gitlab.local",
		Status:        "active",
		Capabilities: []MCPTool{
			{Name: "list_projects", Description: "List projects"},
		},
	}

	server.FromSnapshot(snap)

	if server.ID() != NewMCPServerID("srv-002") {
		t.Errorf("ID 不匹配")
	}
	if server.Code() != "gitlab" {
		t.Errorf("Code 不匹配")
	}
	if server.Name() != "GitLab MCP" {
		t.Errorf("Name 不匹配")
	}
	if server.Status() != "active" {
		t.Errorf("Status 不匹配")
	}
	if len(server.Capabilities()) != 1 {
		t.Errorf("Capabilities 长度不匹配")
	}
}

// AgentMCPBinding tests

func TestNewAgentMCPBinding(t *testing.T) {
	binding := NewAgentMCPBinding(
		NewAgentMCPBindingID("binding-001"),
		NewAgentID("agent-001"),
		NewMCPServerID("srv-001"),
	)

	if binding.ID() != NewAgentMCPBindingID("binding-001") {
		t.Errorf("期望 ID 为 binding-001, 实际为 %s", binding.ID())
	}

	if binding.AgentID() != NewAgentID("agent-001") {
		t.Errorf("期望 AgentID 为 agent-001, 实际为 %s", binding.AgentID())
	}

	if binding.MCPServerID() != NewMCPServerID("srv-001") {
		t.Errorf("期望 MCPServerID 为 srv-001, 实际为 %s", binding.MCPServerID())
	}

	if !binding.IsActive() {
		t.Error("期望默认 IsActive 为 true")
	}

	if binding.AutoLoad() {
		t.Error("期望默认 AutoLoad 为 false")
	}
}

func TestAgentMCPBinding_SetEnabledTools(t *testing.T) {
	binding := NewAgentMCPBinding(
		NewAgentMCPBindingID("binding-001"),
		NewAgentID("agent-001"),
		NewMCPServerID("srv-001"),
	)

	binding.SetEnabledTools([]string{"tool1", "tool2"})

	tools := binding.EnabledTools()
	if len(tools) != 2 {
		t.Errorf("期望 2 个工具, 实际为 %d", len(tools))
	}

	if tools[0] != "tool1" || tools[1] != "tool2" {
		t.Errorf("期望工具为 [tool1, tool2], 实际为 %v", tools)
	}
}

func TestAgentMCPBinding_SetEnabledTools_Empty(t *testing.T) {
	binding := NewAgentMCPBinding(
		NewAgentMCPBindingID("binding-001"),
		NewAgentID("agent-001"),
		NewMCPServerID("srv-001"),
	)

	binding.SetEnabledTools([]string{"tool1"})
	binding.SetEnabledTools([]string{}) // 空切片设为 nil

	if binding.EnabledTools() != nil {
		t.Errorf("期望 EnabledTools 为 nil, 实际为 %v", binding.EnabledTools())
	}
}

func TestAgentMCPBinding_SetActive(t *testing.T) {
	binding := NewAgentMCPBinding(
		NewAgentMCPBindingID("binding-001"),
		NewAgentID("agent-001"),
		NewMCPServerID("srv-001"),
	)

	binding.SetActive(false)

	if binding.IsActive() {
		t.Error("期望 IsActive 为 false")
	}
}

func TestAgentMCPBinding_SetAutoLoad(t *testing.T) {
	binding := NewAgentMCPBinding(
		NewAgentMCPBindingID("binding-001"),
		NewAgentID("agent-001"),
		NewMCPServerID("srv-001"),
	)

	binding.SetAutoLoad(true)

	if !binding.AutoLoad() {
		t.Error("期望 AutoLoad 为 true")
	}
}

func TestAgentMCPBinding_EnabledTools_ReturnsCopy(t *testing.T) {
	binding := NewAgentMCPBinding(
		NewAgentMCPBindingID("binding-001"),
		NewAgentID("agent-001"),
		NewMCPServerID("srv-001"),
	)

	binding.SetEnabledTools([]string{"tool1", "tool2"})

	tools1 := binding.EnabledTools()
	tools1[0] = "modified"

	tools2 := binding.EnabledTools()
	if tools2[0] == "modified" {
		t.Error("EnabledTools 应返回拷贝，不应受外部修改影响")
	}
}

func TestAgentMCPBinding_ToSnapshot(t *testing.T) {
	binding := NewAgentMCPBinding(
		NewAgentMCPBindingID("binding-001"),
		NewAgentID("agent-001"),
		NewMCPServerID("srv-001"),
	)
	binding.SetEnabledTools([]string{"tool1"})
	binding.SetActive(false)
	binding.SetAutoLoad(true)

	snap := binding.ToSnapshot()

	if snap.ID != binding.ID() {
		t.Errorf("ID 不匹配")
	}
	if snap.AgentID != binding.AgentID() {
		t.Errorf("AgentID 不匹配")
	}
	if snap.MCPServerID != binding.MCPServerID() {
		t.Errorf("MCPServerID 不匹配")
	}
	if len(snap.EnabledTools) != 1 {
		t.Errorf("EnabledTools 长度不匹配")
	}
	if snap.IsActive != binding.IsActive() {
		t.Errorf("IsActive 不匹配")
	}
	if snap.AutoLoad != binding.AutoLoad() {
		t.Errorf("AutoLoad 不匹配")
	}
}

func TestAgentMCPBinding_FromSnapshot(t *testing.T) {
	binding := NewAgentMCPBinding(
		NewAgentMCPBindingID("binding-001"),
		NewAgentID("agent-001"),
		NewMCPServerID("srv-001"),
	)

	snap := AgentMCPBindingSnapshot{
		ID:           NewAgentMCPBindingID("binding-002"),
		AgentID:      NewAgentID("agent-002"),
		MCPServerID:  NewMCPServerID("srv-002"),
		EnabledTools: []string{"tool1", "tool2"},
		IsActive:     false,
		AutoLoad:     true,
	}

	binding.FromSnapshot(snap)

	if binding.ID() != NewAgentMCPBindingID("binding-002") {
		t.Errorf("ID 不匹配")
	}
	if binding.AgentID() != NewAgentID("agent-002") {
		t.Errorf("AgentID 不匹配")
	}
	if binding.MCPServerID() != NewMCPServerID("srv-002") {
		t.Errorf("MCPServerID 不匹配")
	}
	if len(binding.EnabledTools()) != 2 {
		t.Errorf("EnabledTools 长度不匹配")
	}
	if binding.IsActive() {
		t.Error("IsActive 应为 false")
	}
	if !binding.AutoLoad() {
		t.Error("AutoLoad 应为 true")
	}
}
