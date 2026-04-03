/**
 * Skill Handler 测试
 */
package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/weibh/taskmanager/infrastructure/skill"
)

// setupTestSkillHandler 创建测试用的 SkillHandler 和路由
func setupTestSkillHandler(tempDir string) (*SkillHandler, *http.ServeMux) {
	loader := skill.NewSkillsLoaderWithPaths([]string{tempDir})
	handler := NewSkillHandler(loader)
	mux := SetupSkillRoutes(handler)
	return handler, mux
}

// SetupSkillRoutes 设置技能路由
func SetupSkillRoutes(handler *SkillHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/skills", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handler.ListSkills(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/v1/skills/detail", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handler.GetSkill(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/v1/skills/simple", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handler.ListSkillsSimple(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	return mux
}

// createTestSkillDir 创建测试用的技能目录结构
func createTestSkillDir(t *testing.T, skillName string, skillContent string) string {
	tempDir, err := os.MkdirTemp("", "skill-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}

	skillDir := filepath.Join(tempDir, skillName)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("创建技能目录失败: %v", err)
	}

	skillFile := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte(skillContent), 0644); err != nil {
		t.Fatalf("创建技能文件失败: %v", err)
	}

	return tempDir
}

// cleanupTestDir 清理测试目录
func cleanupTestDir(t *testing.T, tempDir string) {
	if err := os.RemoveAll(tempDir); err != nil {
		t.Errorf("清理临时目录失败: %v", err)
	}
}

func TestListSkills(t *testing.T) {
	skillContent := `---
description: 测试技能描述
requires_bins: go, git
---
# 测试技能

这是测试技能的内容。
`
	tempDir := createTestSkillDir(t, "test-skill", skillContent)
	defer cleanupTestDir(t, tempDir)

	_, mux := setupTestSkillHandler(tempDir)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("期望 Content-Type 为 'application/json', 实际为 '%s'", contentType)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	items, ok := resp["items"].([]interface{})
	if !ok {
		t.Fatal("响应中 items 字段类型不正确")
	}

	if len(items) != 1 {
		t.Errorf("期望有 1 个技能, 实际有 %d 个", len(items))
	}

	total, ok := resp["total"].(float64)
	if !ok || int(total) != 1 {
		t.Errorf("期望 total 为 1, 实际为 %v", resp["total"])
	}
}

func TestListSkills_Empty(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "skill-test-empty-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer cleanupTestDir(t, tempDir)

	_, mux := setupTestSkillHandler(tempDir)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	items, ok := resp["items"].([]interface{})
	if !ok {
		t.Fatal("响应中 items 字段类型不正确")
	}

	if len(items) != 0 {
		t.Errorf("期望有 0 个技能, 实际有 %d 个", len(items))
	}
}

func TestGetSkill_Success(t *testing.T) {
	skillContent := `---
description: 测试技能详情
requires_bins: go
requires_env: HOME
---
# 测试技能详情

这是测试技能的详细内容。
包含多行文本。
`
	tempDir := createTestSkillDir(t, "test-skill-detail", skillContent)
	defer cleanupTestDir(t, tempDir)

	_, mux := setupTestSkillHandler(tempDir)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills/detail?name=test-skill-detail", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("期望 Content-Type 为 'application/json', 实际为 '%s'", contentType)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp["name"] != "test-skill-detail" {
		t.Errorf("期望 name 为 'test-skill-detail', 实际为 '%v'", resp["name"])
	}

	content, ok := resp["content"].(string)
	if !ok || content == "" {
		t.Error("响应中应该包含非空的 content")
	}

	// 验证 content 不包含 frontmatter
	if len(content) > 0 {
		expectedContent := "# 测试技能详情\n\n这是测试技能的详细内容。\n包含多行文本。"
		if content != expectedContent {
			t.Errorf("期望 content 为 '%s', 实际为 '%s'", expectedContent, content)
		}
	}

	// 验证 metadata
	metadata, ok := resp["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("响应中 metadata 字段类型不正确")
	}

	if metadata["description"] != "测试技能详情" {
		t.Errorf("期望 metadata.description 为 '测试技能详情', 实际为 '%v'", metadata["description"])
	}

	// 验证 available（依赖环境中是否有 go 和 HOME 环境变量）
	available, ok := resp["available"].(bool)
	if !ok {
		t.Error("响应中 available 字段类型不正确")
	}
	// 注意：available 的值取决于当前环境，这里只验证字段存在且类型正确
	t.Logf("available: %v", available)

	// 验证 requires
	requires, _ := resp["requires"].(string)
	// 如果 available 为 false，requires 应该有值
	if !available && requires == "" {
		t.Error("当 available 为 false 时，requires 字段应该有值")
	}
}

func TestGetSkill_MissingName(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "skill-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer cleanupTestDir(t, tempDir)

	_, mux := setupTestSkillHandler(tempDir)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills/detail", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusBadRequest, w.Code)
	}

	body := w.Body.String()
	if body != "skill name is required\n" {
		t.Errorf("期望错误信息为 'skill name is required', 实际为 '%s'", body)
	}
}

func TestGetSkill_NotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "skill-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer cleanupTestDir(t, tempDir)

	_, mux := setupTestSkillHandler(tempDir)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills/detail?name=non-existent-skill", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusNotFound, w.Code)
	}

	body := w.Body.String()
	if body != "skill not found\n" {
		t.Errorf("期望错误信息为 'skill not found', 实际为 '%s'", body)
	}
}

func TestGetSkill_NoMetadata(t *testing.T) {
	// 没有 frontmatter 的技能文件
	skillContent := `# 无元数据技能

这是一个没有 YAML frontmatter 的技能。
`
	tempDir := createTestSkillDir(t, "no-meta-skill", skillContent)
	defer cleanupTestDir(t, tempDir)

	_, mux := setupTestSkillHandler(tempDir)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills/detail?name=no-meta-skill", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp["name"] != "no-meta-skill" {
		t.Errorf("期望 name 为 'no-meta-skill', 实际为 '%v'", resp["name"])
	}

	// 没有 frontmatter 时，metadata 应该为 nil
	if resp["metadata"] != nil {
		t.Errorf("期望 metadata 为 nil, 实际为 %v", resp["metadata"])
	}

	// 没有要求时，available 应该为 true
	if resp["available"] != true {
		t.Errorf("期望 available 为 true, 实际为 %v", resp["available"])
	}
}

func TestListSkillsSimple(t *testing.T) {
	skillContent1 := `---
description: 技能1描述
---
# 技能1

技能1内容。
`
	skillContent2 := `---
description: 技能2描述
---
# 技能2

技能2内容。
`
	tempDir, err := os.MkdirTemp("", "skill-test-simple-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer cleanupTestDir(t, tempDir)

	// 创建两个技能
	skillDir1 := filepath.Join(tempDir, "skill1")
	os.MkdirAll(skillDir1, 0755)
	os.WriteFile(filepath.Join(skillDir1, "SKILL.md"), []byte(skillContent1), 0644)

	skillDir2 := filepath.Join(tempDir, "skill2")
	os.MkdirAll(skillDir2, 0755)
	os.WriteFile(filepath.Join(skillDir2, "SKILL.md"), []byte(skillContent2), 0644)

	_, mux := setupTestSkillHandler(tempDir)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills/simple", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("期望 Content-Type 为 'application/json', 实际为 '%s'", contentType)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	items, ok := resp["items"].([]interface{})
	if !ok {
		t.Fatal("响应中 items 字段类型不正确")
	}

	if len(items) != 2 {
		t.Errorf("期望有 2 个技能, 实际有 %d 个", len(items))
	}

	// 验证每个技能项只包含 name 和 description
	for _, item := range items {
		skillItem, ok := item.(map[string]interface{})
		if !ok {
			t.Error("技能项类型不正确")
			continue
		}

		if _, ok := skillItem["name"]; !ok {
			t.Error("技能项应该包含 name 字段")
		}

		if _, ok := skillItem["description"]; !ok {
			t.Error("技能项应该包含 description 字段")
		}

		// 不应该包含 available、requires 等字段
		if _, ok := skillItem["available"]; ok {
			t.Error("简单列表不应该包含 available 字段")
		}
	}

	total, ok := resp["total"].(float64)
	if !ok || int(total) != 2 {
		t.Errorf("期望 total 为 2, 实际为 %v", resp["total"])
	}
}

func TestListSkillsSimple_Empty(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "skill-test-simple-empty-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer cleanupTestDir(t, tempDir)

	_, mux := setupTestSkillHandler(tempDir)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills/simple", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码为 %d, 实际为 %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	items, ok := resp["items"].([]interface{})
	if !ok {
		t.Fatal("响应中 items 字段类型不正确")
	}

	if len(items) != 0 {
		t.Errorf("期望有 0 个技能, 实际有 %d 个", len(items))
	}

	total, ok := resp["total"].(float64)
	if !ok || int(total) != 0 {
		t.Errorf("期望 total 为 0, 实际为 %v", resp["total"])
	}
}
