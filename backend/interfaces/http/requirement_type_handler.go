package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/domain"
)

type RequirementTypeHandler struct {
	requirementTypeRepo domain.RequirementTypeEntityRepository
}

func NewRequirementTypeHandler(requirementTypeRepo domain.RequirementTypeEntityRepository) *RequirementTypeHandler {
	return &RequirementTypeHandler{
		requirementTypeRepo: requirementTypeRepo,
	}
}

// EnsureDefaultRequirementTypes 确保项目有所需的默认类型（normal, heartbeat）
// 如果类型已存在则跳过，不存在则创建
func (h *RequirementTypeHandler) EnsureDefaultRequirementTypes(ctx context.Context, projectID domain.ProjectID) error {
	defaultTypes := []struct {
		code        string
		name        string
		description string
		color       string
	}{
		{"normal", "普通需求", "普通流程需求，需要人工触发", "blue"},
		{"heartbeat", "心跳需求", "自动触发的心跳任务", "green"},
	}

	for _, dt := range defaultTypes {
		// 检查是否已存在
		existing, err := h.requirementTypeRepo.FindByCode(ctx, projectID, dt.code)
		if err != nil {
			return err
		}
		if existing != nil {
			continue // 已存在，跳过
		}

		// 创建默认类型
		rt, err := domain.NewRequirementTypeEntity(
			domain.NewRequirementTypeEntityID(generateID()),
			projectID,
			dt.code,
			dt.name,
			dt.description,
		)
		if err != nil {
			return err
		}
		rt.SetColor(dt.color)
		if err := h.requirementTypeRepo.Save(ctx, rt); err != nil {
			return err
		}
	}
	return nil
}

// IsBuiltInRequirementType 检查是否为内置需求类型
func IsBuiltInRequirementType(code string) bool {
	return code == "normal" || code == "heartbeat"
}

type CreateRequirementTypeRequest struct {
	ProjectID   string `json:"project_id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Color       string `json:"color"`
}

func (h *RequirementTypeHandler) CreateRequirementType(c *gin.Context) {
	var req CreateRequirementTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	if req.ProjectID == "" || req.Code == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "project_id, code and name are required"})
		return
	}

	rt, err := domain.NewRequirementTypeEntity(
		domain.NewRequirementTypeEntityID(generateID()),
		domain.NewProjectID(req.ProjectID),
		req.Code,
		req.Name,
		req.Description,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}

	if req.Icon != "" {
		rt.SetIcon(req.Icon)
	}
	if req.Color != "" {
		rt.SetColor(req.Color)
	}

	if err := h.requirementTypeRepo.Save(c.Request.Context(), rt); err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, h.requirementTypeToMap(rt))
}

func (h *RequirementTypeHandler) ListRequirementTypes(c *gin.Context) {
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "project_id is required"})
		return
	}

	types, err := h.requirementTypeRepo.FindByProjectID(c.Request.Context(), domain.NewProjectID(projectIDStr))
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	resp := make([]map[string]interface{}, 0, len(types))
	for _, rt := range types {
		resp = append(resp, h.requirementTypeToMap(rt))
	}
	c.JSON(http.StatusOK, resp)
}

func (h *RequirementTypeHandler) DeleteRequirementType(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}

	// 先查询获取类型信息，检查是否为内置类型
	rt, err := h.requirementTypeRepo.FindByID(c.Request.Context(), domain.NewRequirementTypeEntityID(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	if rt == nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "requirement type not found"})
		return
	}

	// 检查是否为内置类型，不允许删除
	if IsBuiltInRequirementType(rt.Code()) {
		c.JSON(http.StatusForbidden, HTTPError{Code: http.StatusForbidden, Message: "cannot delete built-in requirement type"})
		return
	}

	if err := h.requirementTypeRepo.Delete(c.Request.Context(), domain.NewRequirementTypeEntityID(id)); err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, map[string]string{"message": "ok"})
}

func (h *RequirementTypeHandler) requirementTypeToMap(rt *domain.RequirementTypeEntity) map[string]interface{} {
	return map[string]interface{}{
		"id":               rt.ID().String(),
		"project_id":       rt.ProjectID().String(),
		"code":             rt.Code(),
		"name":             rt.Name(),
		"description":      rt.Description(),
		"icon":             rt.Icon(),
		"color":            rt.Color(),
		"sort_order":       rt.SortOrder(),
		"state_machine_id": rt.StateMachineID(),
		"created_at":       rt.CreatedAt().UnixMilli(),
		"updated_at":       rt.UpdatedAt().UnixMilli(),
	}
}

func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
