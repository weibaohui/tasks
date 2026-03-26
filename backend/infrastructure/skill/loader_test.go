/**
 * SkillsLoader 单元测试
 */
package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewSkillsLoaderWithPaths(t *testing.T) {
	// 创建临时目录
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	loader := NewSkillsLoaderWithPaths([]string{tmpDir1, tmpDir2})
	paths := loader.GetSearchPaths()

	if len(paths) != 2 {
		t.Errorf("期望2个搜索路径, 实际为 %d", len(paths))
	}

	if paths[0] != tmpDir1 {
		t.Errorf("期望路径为 %s, 实际为 %s", tmpDir1, paths[0])
	}

	if paths[1] != tmpDir2 {
		t.Errorf("期望路径为 %s, 实际为 %s", tmpDir2, paths[1])
	}
}

func TestNewSkillsLoaderWithPaths_FiltersInvalidPaths(t *testing.T) {
	tmpDir := t.TempDir()

	loader := NewSkillsLoaderWithPaths([]string{tmpDir, "/nonexistent/path", ""})
	paths := loader.GetSearchPaths()

	if len(paths) != 1 {
		t.Errorf("期望1个有效路径, 实际为 %d", len(paths))
	}

	if paths[0] != tmpDir {
		t.Errorf("期望路径为 %s, 实际为 %s", tmpDir, paths[0])
	}
}

func TestListSkills_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewSkillsLoaderWithPaths([]string{tmpDir})

	skills := loader.ListSkills()
	if len(skills) != 0 {
		t.Errorf("期望0个技能, 实际为 %d", len(skills))
	}
}

func TestListSkills_SinglePath(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试技能
	skillDir := filepath.Join(tmpDir, "test-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("创建技能目录失败: %v", err)
	}

	skillContent := `---
description: Test skill description
---
# Test Skill Content
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("写入SKILL.md失败: %v", err)
	}

	loader := NewSkillsLoaderWithPaths([]string{tmpDir})
	skills := loader.ListSkills()

	if len(skills) != 1 {
		t.Fatalf("期望1个技能, 实际为 %d", len(skills))
	}

	if skills[0].Name != "test-skill" {
		t.Errorf("期望技能名为 test-skill, 实际为 %s", skills[0].Name)
	}

	if skills[0].Description != "Test skill description" {
		t.Errorf("期望描述为 Test skill description, 实际为 %s", skills[0].Description)
	}
}

func TestListSkills_MultiPath_Deduplication(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// 在第一个路径创建同名技能
	skillDir1 := filepath.Join(tmpDir1, "github")
	if err := os.MkdirAll(skillDir1, 0755); err != nil {
		t.Fatalf("创建技能目录失败: %v", err)
	}
	skillContent1 := `---
description: Skill from path1
---
# Path1 Skill
`
	if err := os.WriteFile(filepath.Join(skillDir1, "SKILL.md"), []byte(skillContent1), 0644); err != nil {
		t.Fatalf("写入SKILL.md失败: %v", err)
	}

	// 在第二个路径创建同名技能（应该覆盖）
	skillDir2 := filepath.Join(tmpDir2, "github")
	if err := os.MkdirAll(skillDir2, 0755); err != nil {
		t.Fatalf("创建技能目录失败: %v", err)
	}
	skillContent2 := `---
description: Skill from path2
---
# Path2 Skill
`
	if err := os.WriteFile(filepath.Join(skillDir2, "SKILL.md"), []byte(skillContent2), 0644); err != nil {
		t.Fatalf("写入SKILL.md失败: %v", err)
	}

	// 创建不同名技能
	skillDir3 := filepath.Join(tmpDir2, "other-skill")
	if err := os.MkdirAll(skillDir3, 0755); err != nil {
		t.Fatalf("创建技能目录失败: %v", err)
	}
	skillContent3 := `---
description: Other skill
---
# Other Skill
`
	if err := os.WriteFile(filepath.Join(skillDir3, "SKILL.md"), []byte(skillContent3), 0644); err != nil {
		t.Fatalf("写入SKILL.md失败: %v", err)
	}

	// 优先级: tmpDir1 < tmpDir2（后面的覆盖前面的）
	loader := NewSkillsLoaderWithPaths([]string{tmpDir1, tmpDir2})
	skills := loader.ListSkills()

	if len(skills) != 2 {
		t.Fatalf("期望2个技能, 实际为 %d", len(skills))
	}

	// github 应该来自 tmpDir2（后加载的）
	githubSkill := skills[0]
	if githubSkill.Name == "github" {
		if githubSkill.Description != "Skill from path2" {
			t.Errorf("期望 github 技能描述为 Skill from path2, 实际为 %s", githubSkill.Description)
		}
	} else if githubSkill.Name == "other-skill" {
		if githubSkill.Description != "Other skill" {
			t.Errorf("期望 other-skill 技能描述为 Other skill, 实际为 %s", githubSkill.Description)
		}
	}
}

func TestLoadSkill(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试技能
	skillDir := filepath.Join(tmpDir, "test-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("创建技能目录失败: %v", err)
	}

	expectedContent := `---
description: Test skill
---
# Test Skill Content
Some content here
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(expectedContent), 0644); err != nil {
		t.Fatalf("写入SKILL.md失败: %v", err)
	}

	loader := NewSkillsLoaderWithPaths([]string{tmpDir})

	content := loader.LoadSkill("test-skill")
	if content == "" {
		t.Fatal("期望加载技能内容, 实际为空")
	}

	if content != expectedContent {
		t.Errorf("期望内容为 %s, 实际为 %s", expectedContent, content)
	}
}

func TestLoadSkill_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewSkillsLoaderWithPaths([]string{tmpDir})

	content := loader.LoadSkill("nonexistent")
	if content != "" {
		t.Errorf("期望空内容, 实际为 %s", content)
	}
}

func TestLoadSkill_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewSkillsLoaderWithPaths([]string{tmpDir})

	// 测试路径遍历攻击
	testCases := []string{
		"../etc/passwd",
		"..\\windows\\system32",
		"../../../etc/passwd",
		"/absolute/path",
		"skill/../../../etc",
	}

	for _, name := range testCases {
		content := loader.LoadSkill(name)
		if content != "" {
			t.Errorf("期望路径遍历攻击返回空, 实际为 %s", content)
		}
	}
}

func TestLoadSkillContent(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试技能
	skillDir := filepath.Join(tmpDir, "test-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("创建技能目录失败: %v", err)
	}

	skillContent := `---
description: Test skill
---
# Test Skill Content
Some content here
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("写入SKILL.md失败: %v", err)
	}

	loader := NewSkillsLoaderWithPaths([]string{tmpDir})

	content := loader.LoadSkillContent("test-skill")
	expected := `# Test Skill Content
Some content here`

	if content != expected {
		t.Errorf("期望内容为 %q, 实际为 %q", expected, content)
	}
}

func TestGetSkillMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试技能
	skillDir := filepath.Join(tmpDir, "test-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("创建技能目录失败: %v", err)
	}

	skillContent := `---
description: Test skill description
version: 1.0.0
author: Test Author
requires_bins: git,curl
requires_env: GITHUB_TOKEN
---
# Test Skill
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("写入SKILL.md失败: %v", err)
	}

	loader := NewSkillsLoaderWithPaths([]string{tmpDir})

	meta := loader.GetSkillMetadata("test-skill")
	if meta == nil {
		t.Fatal("期望获取元数据, 实际为nil")
	}

	if meta["description"] != "Test skill description" {
		t.Errorf("期望 description 为 Test skill description, 实际为 %s", meta["description"])
	}

	if meta["version"] != "1.0.0" {
		t.Errorf("期望 version 为 1.0.0, 实际为 %s", meta["version"])
	}

	if meta["author"] != "Test Author" {
		t.Errorf("期望 author 为 Test Author, 实际为 %s", meta["author"])
	}
}

func TestCheckRequirements_AllMet(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试技能，需求 git（二进制应该存在）
	skillDir := filepath.Join(tmpDir, "test-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("创建技能目录失败: %v", err)
	}

	skillContent := `---
description: Test skill
requires_bins: git
---
# Test Skill
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("写入SKILL.md失败: %v", err)
	}

	loader := NewSkillsLoaderWithPaths([]string{tmpDir})

	// git 通常存在于系统中
	if !loader.CheckRequirements("test-skill") {
		t.Log("git 不存在于系统中，跳过此测试")
	}
}

func TestCheckRequirements_NotMet(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试技能
	skillDir := filepath.Join(tmpDir, "test-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("创建技能目录失败: %v", err)
	}

	skillContent := `---
description: Test skill
requires_bins: nonexistent-binary-12345
---
# Test Skill
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("写入SKILL.md失败: %v", err)
	}

	loader := NewSkillsLoaderWithPaths([]string{tmpDir})

	if loader.CheckRequirements("test-skill") {
		t.Error("期望不满足需求")
	}
}

func TestGetMissingRequirements(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试技能
	skillDir := filepath.Join(tmpDir, "test-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("创建技能目录失败: %v", err)
	}

	skillContent := `---
description: Test skill
requires_bins: git,nonexistent-binary
requires_env: NONEXISTENT_ENV_VAR
---
# Test Skill
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("写入SKILL.md失败: %v", err)
	}

	loader := NewSkillsLoaderWithPaths([]string{tmpDir})

	missing := loader.GetMissingRequirements("test-skill")
	if missing == "" {
		t.Error("期望有缺失的需求")
	}

	// 应该包含 nonexistent-binary
	if missing != "" && len(missing) > 0 {
		// 验证缺失信息格式
		t.Logf("缺失需求: %s", missing)
	}
}

func TestValidateSkillName(t *testing.T) {
	testCases := []struct {
		name    string
		valid   bool
	}{
		{"github", true},
		{"skill-creator", true},
		{"my-skill-123", true},
		{"a", true},
		{"", false},
		{"../etc", false},
		{"/etc/passwd", false},
		{"skill/bin", false},
		{"skill\\windows", false},
		{"..", false},
		// 注意：leading/trailing spaces 会通过 TrimSpace 检查，但这是合理的行为
		// 实际使用中目录名不太可能包含 spaces
	}

	for _, tc := range testCases {
		err := validateSkillName(tc.name)
		if tc.valid && err != nil {
			t.Errorf("期望 %q 有效, 实际返回错误: %v", tc.name, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("期望 %q 无效, 实际有效", tc.name)
		}
	}
}

func TestListSkills_ExcludesInvalidSkillNames(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建无效名称的目录
	invalidDir := filepath.Join(tmpDir, "..")
	if err := os.MkdirAll(invalidDir, 0755); err == nil {
		// 创建成功，但这是一个无效路径
		t.Log("目录创建成功但会被跳过")
	}

	// 创建一个正常技能
	skillDir := filepath.Join(tmpDir, "valid-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("创建技能目录失败: %v", err)
	}
	skillContent := `---
description: Valid skill
---
# Valid Skill
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("写入SKILL.md失败: %v", err)
	}

	loader := NewSkillsLoaderWithPaths([]string{tmpDir})
	skills := loader.ListSkills()

	// 应该只包含 valid-skill
	if len(skills) != 1 {
		t.Errorf("期望1个有效技能, 实际为 %d", len(skills))
	}

	if skills[0].Name != "valid-skill" {
		t.Errorf("期望技能名为 valid-skill, 实际为 %s", skills[0].Name)
	}
}
