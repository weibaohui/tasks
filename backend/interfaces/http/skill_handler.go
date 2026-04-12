/**
 * Skill HTTP Handler
 * 处理技能相关的 HTTP 请求
 */
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/domain"
)

// SkillHandler 技能处理器
type SkillHandler struct {
	loader domain.SkillsLoader
}

// NewSkillHandler 创建技能处理器
func NewSkillHandler(loader domain.SkillsLoader) *SkillHandler {
	return &SkillHandler{
		loader: loader,
	}
}

// ListSkills 列出所有技能
func (h *SkillHandler) ListSkills(c *gin.Context) {
	skills := h.loader.ListSkills()

	c.JSON(http.StatusOK, map[string]interface{}{
		"items": skills,
		"total": len(skills),
	})
}

// GetSkill 获取单个技能详情
func (h *SkillHandler) GetSkill(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "skill name is required"})
		return
	}

	content := h.loader.LoadSkillContent(name)
	if content == "" {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "skill not found"})
		return
	}

	metadata := h.loader.GetSkillMetadata(name)
	available := h.loader.CheckRequirements(name)
	missing := ""
	if !available {
		missing = h.loader.GetMissingRequirements(name)
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"name":      name,
		"content":   content,
		"metadata":  metadata,
		"available": available,
		"requires":  missing,
	})
}

// ListSkillsSimple 获取所有技能名称列表（简单版，用于下拉选择）
func (h *SkillHandler) ListSkillsSimple(c *gin.Context) {
	skills := h.loader.ListSkills()

	result := make([]map[string]string, 0, len(skills))
	for _, s := range skills {
		result = append(result, map[string]string{
			"name":        s.Name,
			"description": s.Description,
		})
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"items": result,
		"total": len(result),
	})
}
