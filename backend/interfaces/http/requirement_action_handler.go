package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type UpdateRequirementStatusRequest struct {
	ID        string `json:"id"`
	NewStatus string `json:"new_status"`
}

func (h *RequirementHandler) UpdateRequirementStatus(c *gin.Context) {
	var req UpdateRequirementStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	requirement, err := h.requirementService.UpdateRequirementStatus(c.Request.Context(), application.UpdateRequirementStatusCommand{
		ID:        domain.NewRequirementID(req.ID),
		NewStatus: req.NewStatus,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.requirementToMap(requirement))
}

func (h *RequirementHandler) GetRequirementTransitionHistory(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	history, err := h.requirementService.GetRequirementTransitionHistory(c.Request.Context(), domain.NewRequirementID(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	// 转换为响应格式
	resp := make([]map[string]interface{}, 0, len(history))
	for _, log := range history {
		resp = append(resp, map[string]interface{}{
			"id":             log.ID,
			"requirement_id": log.RequirementID,
			"from_state":     log.FromState,
			"to_state":       log.ToState,
			"trigger":        log.Trigger,
			"triggered_by":   log.TriggeredBy,
			"remark":         log.Remark,
			"result":         log.Result,
			"error_message":  log.ErrorMessage,
			"created_at":     log.CreatedAt.UnixMilli(),
		})
	}
	c.JSON(http.StatusOK, resp)
}

func (h *RequirementHandler) GetStatusStats(c *gin.Context) {
	projectIDStr := c.Query("project_id")
	var projectID *domain.ProjectID
	if projectIDStr != "" {
		id := domain.NewProjectID(projectIDStr)
		projectID = &id
	}

	stats, err := h.requirementService.GetStatusStats(c.Request.Context(), projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *RequirementHandler) DeleteRequirement(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	err := h.requirementService.DeleteRequirement(c.Request.Context(), application.DeleteRequirementCommand{
		ID: domain.NewRequirementID(id),
	})
	if err != nil {
		h.handleDeleteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *RequirementHandler) BatchDeleteRequirements(c *gin.Context) {
	var req BatchDeleteRequirementsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "ids is required"})
		return
	}
	ids := make([]domain.RequirementID, 0, len(req.IDs))
	for _, id := range req.IDs {
		ids = append(ids, domain.NewRequirementID(id))
	}
	err := h.requirementService.BatchDeleteRequirements(c.Request.Context(), application.BatchDeleteRequirementsCommand{
		IDs: ids,
	})
	if err != nil {
		h.handleDeleteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// - ErrRequirementNotFound -> 404
// - 上下文取消/超时 -> 500
// - 其他内部错误 -> 500（不暴露原始错误信息）
func (h *RequirementHandler) handleDeleteError(c *gin.Context, err error) {
	if errors.Is(err, application.ErrRequirementNotFound) {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "requirement not found"})
		return
	}

	// 检查上下文错误
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "request timeout or cancelled"})
		return
	}

	// 其他内部错误，不暴露原始错误信息
	c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "internal server error"})
}
