/**
 * Skill HTTP Handler
 * 处理技能相关的 HTTP 请求
 */
package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/weibh/taskmanager/infrastructure/skill"
)

// SkillHandler 技能处理器
type SkillHandler struct {
	loader *skill.SkillsLoader
}

// SkillListItem 安全技能列表项（不包含内部路径）
type SkillListItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
	Available   bool   `json:"available"`
	Requires    string `json:"requires,omitempty"`
}

// NewSkillHandler 创建技能处理器
func NewSkillHandler(loader *skill.SkillsLoader) *SkillHandler {
	return &SkillHandler{
		loader: loader,
	}
}

// ListSkills 列出所有技能
func (h *SkillHandler) ListSkills(w http.ResponseWriter, r *http.Request) {
	skills := h.loader.ListSkills()

	// 转换为安全列表项，不暴露内部路径
	items := make([]SkillListItem, 0, len(skills))
	for _, s := range skills {
		items = append(items, SkillListItem{
			Name:        s.Name,
			Description: s.Description,
			Source:      s.Source,
			Available:   s.Available,
			Requires:    s.Requires,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": items,
		"total": len(items),
	})
}

// GetSkill 获取单个技能详情
func (h *SkillHandler) GetSkill(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "skill name is required", http.StatusBadRequest)
		return
	}

	content := h.loader.LoadSkillContent(name)
	if content == "" {
		http.Error(w, "skill not found", http.StatusNotFound)
		return
	}

	metadata := h.loader.GetSkillMetadata(name)
	available := h.loader.CheckRequirements(name)
	missing := ""
	if !available {
		missing = h.loader.GetMissingRequirements(name)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":        name,
		"content":     content,
		"metadata":    metadata,
		"available":   available,
		"requires":    missing,
		"source":      h.getSkillSource(name),
	})
}

// getSkillSource 获取技能来源
func (h *SkillHandler) getSkillSource(name string) string {
	// 验证技能名称，防止路径遍历攻击
	if err := validateSkillName(name); err != nil {
		return ""
	}

	// 检查工作区
	workspaceSkill := filepath.Join(h.loader.GetWorkspaceSkills(), name, "SKILL.md")
	if _, err := os.Stat(workspaceSkill); err == nil {
		return "workspace"
	}

	// 内置技能
	return "builtin"
}

// validateSkillName 验证技能名称，防止路径遍历攻击
func validateSkillName(name string) error {
	// 检查是否包含路径分隔符
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("skill name contains invalid path characters")
	}

	// 检查是否包含父目录引用
	if strings.Contains(name, "..") {
		return fmt.Errorf("skill name contains parent directory reference")
	}

	// 检查是否为空
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("skill name cannot be empty")
	}

	return nil
}

// ListSkillsSimple 获取所有技能名称列表（简单版，用于下拉选择）
func (h *SkillHandler) ListSkillsSimple(w http.ResponseWriter, r *http.Request) {
	skills := h.loader.ListSkills()

	result := make([]map[string]string, 0)
	for _, s := range skills {
		result = append(result, map[string]string{
			"name":        s.Name,
			"description": s.Description,
			"source":      s.Source,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": result,
		"total": len(result),
	})
}