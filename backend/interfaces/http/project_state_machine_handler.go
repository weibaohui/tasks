package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/application"
)

// ProjectStateMachineHandler 项目状态机 HTTP 处理
type ProjectStateMachineHandler struct {
	service *application.ProjectStateMachineApplicationService
}

// NewProjectStateMachineHandler 创建 handler
func NewProjectStateMachineHandler(service *application.ProjectStateMachineApplicationService) *ProjectStateMachineHandler {
	return &ProjectStateMachineHandler{service: service}
}

// SetProjectStateMachineRequest 设置项目状态机请求
type SetProjectStateMachineRequest struct {
	RequirementType string `json:"requirement_type"`
	StateMachineID  string `json:"state_machine_id"`
}

// ProjectStateMachineResponse 项目状态机响应
type ProjectStateMachineResponse struct {
	ID               string `json:"id"`
	RequirementType  string `json:"requirement_type"`
	StateMachineID   string `json:"state_machine_id"`
	StateMachineName string `json:"state_machine_name,omitempty"`
	CreatedAt        int64  `json:"created_at"`
	UpdatedAt        int64  `json:"updated_at"`
}

// ListProjectStateMachines 列出项目的所有状态机映射
func (h *ProjectStateMachineHandler) ListProjectStateMachines(c *gin.Context) {
	projectID := c.Param("project_id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "project_id is required"})
		return
	}

	mappings, err := h.service.GetProjectStateMachines(c.Request.Context(), application.GetProjectStateMachinesQuery{
		ProjectID: projectID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, mappings)
}

// SetProjectStateMachine 设置项目状态机映射
func (h *ProjectStateMachineHandler) SetProjectStateMachine(c *gin.Context) {
	projectID := c.Param("project_id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "project_id is required"})
		return
	}

	var req SetProjectStateMachineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request body"})
		return
	}

	if req.RequirementType == "" || req.StateMachineID == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "requirement_type and state_machine_id are required"})
		return
	}

	mapping, err := h.service.SetProjectStateMachine(c.Request.Context(), application.SetProjectStateMachineCommand{
		ProjectID:       projectID,
		RequirementType: req.RequirementType,
		StateMachineID:  req.StateMachineID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, mapping)
}

// DeleteProjectStateMachine 删除项目状态机映射
func (h *ProjectStateMachineHandler) DeleteProjectStateMachine(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}

	if err := h.service.DeleteProjectStateMachine(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetProjectStateMachineByType 获取指定类型的项目状态机映射
func (h *ProjectStateMachineHandler) GetProjectStateMachineByType(c *gin.Context) {
	projectID := c.Param("project_id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "project_id is required"})
		return
	}

	requirementType := c.Param("requirement_type")
	if requirementType == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "requirement_type is required"})
		return
	}

	mapping, err := h.service.GetProjectStateMachineByType(c.Request.Context(), projectID, requirementType)
	if err != nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, mapping)
}

// GetAvailableRequirementTypes 获取可用的需求类型列表
func (h *ProjectStateMachineHandler) GetAvailableRequirementTypes(c *gin.Context) {
	types := h.service.GetAvailableRequirementTypes()
	c.JSON(http.StatusOK, map[string]interface{}{
		"types": types,
	})
}
