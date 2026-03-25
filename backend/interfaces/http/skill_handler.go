/**
 * Skill HTTP Handler
 * 处理技能相关的 HTTP 请求
 */
package http

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/weibh/taskmanager/infrastructure/skill"
)

// SkillHandler 技能处理器
type SkillHandler struct {
	loader *skill.SkillsLoader
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": skills,
		"total": len(skills),
	})
}

// GetSkill 获取单个技能详情
func (h *SkillHandler) GetSkill(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "skill name is required", http.StatusBadRequest)
		return
	}

	content := h.loader.LoadSkill(name)
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
	// 检查工作区
	workspaceSkill := filepath.Join(h.loader.GetWorkspaceSkills(), name, "SKILL.md")
	if _, err := os.Stat(workspaceSkill); err == nil {
		return "workspace"
	}

	// 内置技能
	return "builtin"
}

// ListSkillsSimple 获取所有技能名称列表（简单版，用于下拉选择）
func (h *SkillHandler) ListSkillsSimple(w http.ResponseWriter, r *http.Request) {
	skills := h.loader.ListSkills()

	var result []map[string]string
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