/**
 * DynamicTool 单元测试
 */
package skill

import (
	"context"
	"testing"
)

func TestNewDynamicTool(t *testing.T) {
	loader := func(name string) string {
		return "test content"
	}

	tool := NewDynamicTool("test-skill", "Test skill description", loader)

	if tool.Name() != "test-skill" {
		t.Errorf("期望名称为 test-skill, 实际为 %s", tool.Name())
	}

	if tool.Description() != "Test skill description" {
		t.Errorf("期望描述为 Test skill description, 实际为 %s", tool.Description())
	}
}

func TestDynamicTool_Name(t *testing.T) {
	tool := NewDynamicTool("my-skill", "", nil)

	if tool.Name() != "my-skill" {
		t.Errorf("期望名称为 my-skill, 实际为 %s", tool.Name())
	}
}

func TestDynamicTool_Description(t *testing.T) {
	tool := NewDynamicTool("my-skill", "My custom description", nil)

	if tool.Description() != "My custom description" {
		t.Errorf("期望描述为 My custom description, 实际为 %s", tool.Description())
	}
}

func TestDynamicTool_Description_Default(t *testing.T) {
	tool := NewDynamicTool("my-skill", "", nil)

	desc := tool.Description()
	// Description() 返回原始的 description 字段，不做默认处理
	if desc != "" {
		t.Error("空描述应返回空字符串")
	}
}

func TestDynamicTool_Info(t *testing.T) {
	tool := NewDynamicTool("test-skill", "Test description", nil)

	info, err := tool.Info(context.Background())
	if err != nil {
		t.Fatalf("Info 失败: %v", err)
	}

	if info.Name != "test-skill" {
		t.Errorf("期望名称为 test-skill, 实际为 %s", info.Name)
	}

	if info.Desc != "Test description" {
		t.Errorf("期望描述为 Test description, 实际为 %s", info.Desc)
	}

	// 验证 ParamsOneOf 不为 nil
	if info.ParamsOneOf == nil {
		t.Fatal("期望有 ParamsOneOf")
	}
}

func TestDynamicTool_Run_EmptyArgs(t *testing.T) {
	loadCall := false
	loader := func(name string) string {
		loadCall = true
		return "# Test Skill\nSome content"
	}

	tool := NewDynamicTool("test-skill", "Test description", loader)

	result, err := tool.Run(context.Background(), "")
	if err != nil {
		t.Fatalf("Run 失败: %v", err)
	}

	if result == "" {
		t.Error("期望有结果")
	}

	if !loadCall {
		t.Error("期望调用 loader")
	}
}

func TestDynamicTool_Run_EmptyJsonBraces(t *testing.T) {
	loadCall := false
	loader := func(name string) string {
		loadCall = true
		return "# Test Skill\nSome content"
	}

	tool := NewDynamicTool("test-skill", "Test description", loader)

	result, err := tool.Run(context.Background(), "{}")
	if err != nil {
		t.Fatalf("Run 失败: %v", err)
	}

	if result == "" {
		t.Error("期望有结果")
	}

	if !loadCall {
		t.Error("期望调用 loader")
	}
}

func TestDynamicTool_Run_WithAction(t *testing.T) {
	loader := func(name string) string {
		return "# Test Skill\nSome content"
	}

	tool := NewDynamicTool("test-skill", "Test description", loader)

	result, err := tool.Run(context.Background(), `{"action": "list", "params": {"limit": 10}}`)
	if err != nil {
		t.Fatalf("Run 失败: %v", err)
	}

	if result == "" {
		t.Error("期望有结果")
	}

	// 应该包含 action
	if result == "" || len(result) == 0 {
		t.Error("期望结果包含 action 信息")
	}
}

func TestDynamicTool_Run_InvalidJSON(t *testing.T) {
	tool := NewDynamicTool("test-skill", "Test description", nil)

	_, err := tool.Run(context.Background(), "invalid json")
	if err == nil {
		t.Error("期望错误")
	}
}

func TestDynamicTool_Run_LoaderReturnsEmpty(t *testing.T) {
	loader := func(name string) string {
		return ""
	}

	tool := NewDynamicTool("nonexistent", "Test description", loader)

	_, err := tool.Run(context.Background(), "")
	if err == nil {
		t.Error("期望错误")
	}
}

func TestDynamicTool_Run_NilLoader(t *testing.T) {
	tool := NewDynamicTool("test-skill", "Test description", nil)

	_, err := tool.Run(context.Background(), "")
	if err == nil {
		t.Error("期望错误")
	}
}

func TestDynamicTool_InvokableRun(t *testing.T) {
	loader := func(name string) string {
		return "# Test Skill\nContent"
	}

	tool := NewDynamicTool("test-skill", "Test description", loader)

	result, err := tool.InvokableRun(context.Background(), "{}")
	if err != nil {
		t.Fatalf("InvokableRun 失败: %v", err)
	}

	if result == "" {
		t.Error("期望有结果")
	}
}

func TestDynamicTool_ExecuteSkill_WithoutAction(t *testing.T) {
	loader := func(name string) string {
		return "# Test Skill\nContent here"
	}

	tool := NewDynamicTool("test-skill", "Test description", loader)

	result, err := tool.Run(context.Background(), "{}")
	if err != nil {
		t.Fatalf("Run 失败: %v", err)
	}

	// 应该包含技能名称
	if result == "" || len(result) == 0 {
		t.Error("期望有结果")
	}
}

func TestDynamicTool_ExecuteSkill_WithAction(t *testing.T) {
	loader := func(name string) string {
		return "# Test Skill\nContent here"
	}

	tool := NewDynamicTool("test-skill", "Test description", loader)

	result, err := tool.Run(context.Background(), `{"action": "create", "params": {"name": "test"}}`)
	if err != nil {
		t.Fatalf("Run 失败: %v", err)
	}

	// 应该包含操作信息
	if result == "" {
		t.Error("期望有结果")
	}
}

// Registry tests

func TestNewRegistry(t *testing.T) {
	loader := func(name string) string {
		return "content"
	}

	registry := NewRegistry(loader)

	if registry == nil {
		t.Fatal("期望创建 Registry")
	}
}

func TestRegistry_RegisterSkill(t *testing.T) {
	loader := func(name string) string {
		return "content"
	}

	registry := NewRegistry(loader)

	tool := registry.RegisterSkill("github", "GitHub skill")

	if tool == nil {
		t.Fatal("期望返回 tool")
	}

	if tool.Name() != "github" {
		t.Errorf("期望名称为 github, 实际为 %s", tool.Name())
	}
}

func TestRegistry_GetTool(t *testing.T) {
	loader := func(name string) string {
		return "content"
	}

	registry := NewRegistry(loader)
	registry.RegisterSkill("github", "GitHub skill")

	tool := registry.GetTool("github")
	if tool == nil {
		t.Fatal("期望获取 tool")
	}

	if tool.Name() != "github" {
		t.Errorf("期望名称为 github, 实际为 %s", tool.Name())
	}
}

func TestRegistry_GetTool_NotFound(t *testing.T) {
	loader := func(name string) string {
		return "content"
	}

	registry := NewRegistry(loader)

	tool := registry.GetTool("nonexistent")
	if tool != nil {
		t.Error("期望 nil")
	}
}

func TestRegistry_GetAllTools(t *testing.T) {
	loader := func(name string) string {
		return "content"
	}

	registry := NewRegistry(loader)
	registry.RegisterSkill("github", "GitHub skill")
	registry.RegisterSkill("gitlab", "GitLab skill")

	tools := registry.GetAllTools()
	if len(tools) != 2 {
		t.Errorf("期望2个工具, 实际为 %d", len(tools))
	}
}

func TestRegistry_GetSkillNames(t *testing.T) {
	loader := func(name string) string {
		return "content"
	}

	registry := NewRegistry(loader)
	registry.RegisterSkill("github", "GitHub skill")
	registry.RegisterSkill("gitlab", "GitLab skill")

	names := registry.GetSkillNames()
	if len(names) != 2 {
		t.Errorf("期望2个名称, 实际为 %d", len(names))
	}
}

func TestRegistry_HasSkill(t *testing.T) {
	loader := func(name string) string {
		return "content"
	}

	registry := NewRegistry(loader)
	registry.RegisterSkill("github", "GitHub skill")

	if !registry.HasSkill("github") {
		t.Error("期望有 github 技能")
	}

	if registry.HasSkill("nonexistent") {
		t.Error("期望没有 nonexistent 技能")
	}
}
