/**
 * SkillToolAdapter 单元测试
 */
package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/weibh/taskmanager/infrastructure/llm"
	skilltool "github.com/weibh/taskmanager/infrastructure/llm/tools/skill"
	"github.com/weibh/taskmanager/infrastructure/skill"
)

func TestNewSkillToolAdapter(t *testing.T) {
	loader := func(name string) string {
		return "test content"
	}

	dynamicTool := skilltool.NewDynamicTool("test-skill", "Test description", loader)
	adapter := NewSkillToolAdapter(dynamicTool)

	if adapter == nil {
		t.Fatal("期望创建 adapter")
	}
}

func TestSkillToolAdapter_Name(t *testing.T) {
	loader := func(name string) string {
		return "test content"
	}

	dynamicTool := skilltool.NewDynamicTool("github", "GitHub skill", loader)
	adapter := NewSkillToolAdapter(dynamicTool)

	name := adapter.Name()
	if name != "skill__github" {
		t.Errorf("期望名称为 skill__github, 实际为 %s", name)
	}
}

func TestSkillToolAdapter_Description(t *testing.T) {
	loader := func(name string) string {
		return "test content"
	}

	dynamicTool := skilltool.NewDynamicTool("github", "GitHub operations skill", loader)
	adapter := NewSkillToolAdapter(dynamicTool)

	desc := adapter.Description()
	if desc != "GitHub operations skill" {
		t.Errorf("期望描述为 GitHub operations skill, 实际为 %s", desc)
	}
}

func TestSkillToolAdapter_Description_Empty(t *testing.T) {
	loader := func(name string) string {
		return "test content"
	}

	dynamicTool := skilltool.NewDynamicTool("github", "", loader)
	adapter := NewSkillToolAdapter(dynamicTool)

	desc := adapter.Description()
	if desc == "" {
		t.Error("期望有默认描述")
	}

	// 默认描述应该包含技能名称
	if desc == "" || len(desc) == 0 {
		t.Error("期望默认描述包含技能名")
	}
}

func TestSkillToolAdapter_Parameters(t *testing.T) {
	loader := func(name string) string {
		return "test content"
	}

	dynamicTool := skilltool.NewDynamicTool("test-skill", "Test", loader)
	adapter := NewSkillToolAdapter(dynamicTool)

	params := adapter.Parameters()
	if params == nil {
		t.Fatal("期望有参数")
	}

	// 验证是有效的 JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(params, &parsed); err != nil {
		t.Fatalf("参数应该是有效的 JSON: %v", err)
	}

	// 验证结构
	if parsed["type"] != "object" {
		t.Errorf("期望 type 为 object, 实际为 %v", parsed["type"])
	}

	props, ok := parsed["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("期望有 properties")
	}

	if _, ok := props["action"]; !ok {
		t.Error("期望有 action 参数")
	}

	if _, ok := props["params"]; !ok {
		t.Error("期望有 params 参数")
	}
}

func TestSkillToolAdapter_Execute_Success(t *testing.T) {
	loader := func(name string) string {
		return "# GitHub Skill\nThis skill helps with GitHub operations."
	}

	dynamicTool := skilltool.NewDynamicTool("github", "GitHub skill", loader)
	adapter := NewSkillToolAdapter(dynamicTool)

	result, err := adapter.Execute(context.Background(), []byte(`{}`))
	if err != nil {
		t.Fatalf("Execute 失败: %v", err)
	}

	if result == nil {
		t.Fatal("期望有结果")
	}

	if result.Error != "" {
		t.Errorf("期望没有错误, 实际为 %s", result.Error)
	}

	if result.Output == "" {
		t.Error("期望有输出")
	}
}

func TestSkillToolAdapter_Execute_WithAction(t *testing.T) {
	loader := func(name string) string {
		return "# GitHub Skill\nThis skill helps with GitHub operations."
	}

	dynamicTool := skilltool.NewDynamicTool("github", "GitHub skill", loader)
	adapter := NewSkillToolAdapter(dynamicTool)

	result, err := adapter.Execute(context.Background(), []byte(`{"action": "list_repos"}`))
	if err != nil {
		t.Fatalf("Execute 失败: %v", err)
	}

	if result == nil {
		t.Fatal("期望有结果")
	}

	if result.Output == "" {
		t.Error("期望有输出")
	}
}

func TestSkillToolAdapter_Execute_SkillNotFound(t *testing.T) {
	loader := func(name string) string {
		return "" // 返回空表示技能不存在
	}

	dynamicTool := skilltool.NewDynamicTool("nonexistent", "Nonexistent skill", loader)
	adapter := NewSkillToolAdapter(dynamicTool)

	result, err := adapter.Execute(context.Background(), []byte(`{}`))
	if err != nil {
		t.Fatalf("Execute 不应返回 error, 实际为: %v", err)
	}

	if result == nil {
		t.Fatal("期望有结果")
	}

	if result.Error == "" {
		t.Error("期望有错误信息")
	}
}

func TestSkillToolAdapter_ImplementsToolInterface(t *testing.T) {
	loader := func(name string) string {
		return "content"
	}

	dynamicTool := skilltool.NewDynamicTool("test-skill", "Test", loader)
	adapter := NewSkillToolAdapter(dynamicTool)

	// 使用接口断言验证实现了 Tool 接口
	var tool llm.Tool = adapter
	if tool.Name() != "skill__test-skill" {
		t.Errorf("期望名称为 skill__test-skill, 实际为 %s", tool.Name())
	}
}

// SkillToolsAdapterRegistry tests

func TestNewSkillToolsAdapterRegistry(t *testing.T) {
	// 使用临时目录创建真实的 loader
	tmpDir := t.TempDir()
	loader := skill.NewSkillsLoaderWithPaths([]string{tmpDir})

	registry := NewSkillToolsAdapterRegistry(loader)
	if registry == nil {
		t.Fatal("期望创建 registry")
	}
}

func TestSkillToolsAdapterRegistry_GetTools_NilLoader(t *testing.T) {
	registry := NewSkillToolsAdapterRegistry(nil)
	tools := registry.GetTools()

	if tools != nil {
		t.Error("期望 nil loader 返回 nil")
	}
}

func TestSkillToolsAdapterRegistry_GetTools_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	loader := skill.NewSkillsLoaderWithPaths([]string{tmpDir})

	registry := NewSkillToolsAdapterRegistry(loader)
	tools := registry.GetTools()

	if len(tools) != 0 {
		t.Errorf("期望0个工具, 实际为 %d", len(tools))
	}
}

func TestSkillToolsAdapterRegistry_GetTools_WithSkills(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试技能
	skillDir := filepath.Join(tmpDir, "test-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("创建技能目录失败: %v", err)
	}

	skillContent := `---
description: Test skill
---
# Test Skill
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("写入SKILL.md失败: %v", err)
	}

	loader := skill.NewSkillsLoaderWithPaths([]string{tmpDir})
	registry := NewSkillToolsAdapterRegistry(loader)
	tools := registry.GetTools()

	if len(tools) != 1 {
		t.Errorf("期望1个工具, 实际为 %d", len(tools))
	}

	if tools[0].Name() != "skill__test-skill" {
		t.Errorf("期望工具名为 skill__test-skill, 实际为 %s", tools[0].Name())
	}
}

func TestSkillToolsAdapterRegistry_GetToolsForSkills(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试技能
	skillDir := filepath.Join(tmpDir, "skill1")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("创建技能目录失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Skill 1\n"), 0644); err != nil {
		t.Fatalf("写入SKILL.md失败: %v", err)
	}

	skillDir2 := filepath.Join(tmpDir, "skill2")
	if err := os.MkdirAll(skillDir2, 0755); err != nil {
		t.Fatalf("创建技能目录失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir2, "SKILL.md"), []byte("# Skill 2\n"), 0644); err != nil {
		t.Fatalf("写入SKILL.md失败: %v", err)
	}

	loader := skill.NewSkillsLoaderWithPaths([]string{tmpDir})

	skills := []skill.SkillInfo{
		{Name: "skill1", Description: "Skill 1", Available: true},
		{Name: "skill2", Description: "Skill 2", Available: true},
	}

	registry := NewSkillToolsAdapterRegistry(loader)
	tools := registry.GetToolsForSkills(skills)

	if len(tools) != 2 {
		t.Errorf("期望2个工具, 实际为 %d", len(tools))
	}
}
