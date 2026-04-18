package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type HeartbeatHandler struct {
	service        *application.HeartbeatApplicationService
	scheduler      *application.HeartbeatScheduler
	triggerService *application.HeartbeatTriggerService
}

func NewHeartbeatHandler(service *application.HeartbeatApplicationService, scheduler *application.HeartbeatScheduler) *HeartbeatHandler {
	return &HeartbeatHandler{service: service, scheduler: scheduler}
}

func NewHeartbeatHandlerWithTrigger(service *application.HeartbeatApplicationService, scheduler *application.HeartbeatScheduler, triggerService *application.HeartbeatTriggerService) *HeartbeatHandler {
	return &HeartbeatHandler{service: service, scheduler: scheduler, triggerService: triggerService}
}

type CreateHeartbeatRequest struct {
	ProjectID       string `json:"project_id" binding:"required"`
	Name            string `json:"name" binding:"required"`
	IntervalMinutes int    `json:"interval_minutes" binding:"required,min=1"`
	MDContent       string `json:"md_content"`
	AgentCode       string `json:"agent_code"`
	RequirementType string `json:"requirement_type"`
}

type UpdateHeartbeatRequest struct {
	Name            string `json:"name" binding:"required"`
	IntervalMinutes int    `json:"interval_minutes" binding:"required,min=1"`
	MDContent       string `json:"md_content"`
	AgentCode       string `json:"agent_code"`
	RequirementType string `json:"requirement_type"`
	Enabled         bool   `json:"enabled"`
}

func (h *HeartbeatHandler) ListHeartbeats(c *gin.Context) {
	projectID := c.Query("project_id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "project_id is required"})
		return
	}
	heartbeats, err := h.service.ListHeartbeatsByProject(c.Request.Context(), projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(heartbeats))
	for _, hb := range heartbeats {
		resp = append(resp, heartbeatToMap(hb))
	}
	c.JSON(http.StatusOK, resp)
}

func (h *HeartbeatHandler) CreateHeartbeat(c *gin.Context) {
	var req CreateHeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	hb, err := h.service.CreateHeartbeat(c.Request.Context(), application.CreateHeartbeatCommand{
		ProjectID:       req.ProjectID,
		Name:            req.Name,
		IntervalMinutes: req.IntervalMinutes,
		MDContent:       req.MDContent,
		AgentCode:       req.AgentCode,
		RequirementType: req.RequirementType,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, heartbeatToMap(hb))
}

func (h *HeartbeatHandler) GetHeartbeat(c *gin.Context) {
	id := c.Param("id")
	hb, err := h.service.GetHeartbeat(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	if hb == nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "heartbeat not found"})
		return
	}
	c.JSON(http.StatusOK, heartbeatToMap(hb))
}

func (h *HeartbeatHandler) UpdateHeartbeat(c *gin.Context) {
	id := c.Param("id")
	var req UpdateHeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	hb, err := h.service.UpdateHeartbeat(c.Request.Context(), application.UpdateHeartbeatCommand{
		ID:              id,
		Name:            req.Name,
		IntervalMinutes: req.IntervalMinutes,
		MDContent:       req.MDContent,
		AgentCode:       req.AgentCode,
		RequirementType: req.RequirementType,
		Enabled:         req.Enabled,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, heartbeatToMap(hb))
}

func (h *HeartbeatHandler) DeleteHeartbeat(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteHeartbeat(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"message": "ok"})
}

func (h *HeartbeatHandler) TriggerHeartbeat(c *gin.Context) {
	id := c.Param("id")
	if h.triggerService == nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "trigger service not available"})
		return
	}
	if _, err := h.triggerService.TriggerWithSource(c.Request.Context(), id, application.HeartbeatTriggerSourceManual); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"message": "triggered"})
}

// ListHeartbeatRuns 查询指定心跳的最近执行记录。
func (h *HeartbeatHandler) ListHeartbeatRuns(c *gin.Context) {
	id := c.Param("id")
	if h.triggerService == nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "trigger service not available"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	records, err := h.triggerService.ListRunsByHeartbeat(c.Request.Context(), id, limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, records)
}

// ListProjectHeartbeatRuns 查询项目下所有心跳的最近执行记录（聚合视图）。
func (h *HeartbeatHandler) ListProjectHeartbeatRuns(c *gin.Context) {
	projectID := c.Param("project_id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "project_id is required"})
		return
	}
	if h.triggerService == nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "trigger service not available"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	statusQuery := strings.TrimSpace(c.Query("statuses"))
	statuses := make([]string, 0)
	if statusQuery != "" {
		for _, item := range strings.Split(statusQuery, ",") {
			status := strings.TrimSpace(item)
			if status != "" {
				statuses = append(statuses, status)
			}
		}
	}
	page, err := h.triggerService.ListRunsByProject(c.Request.Context(), projectID, limit, offset, statuses)
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, page)
}

func heartbeatToMap(hb *domain.Heartbeat) map[string]interface{} {
	return map[string]interface{}{
		"id":               hb.ID().String(),
		"project_id":       hb.ProjectID().String(),
		"name":             hb.Name(),
		"enabled":          hb.Enabled(),
		"interval_minutes": hb.IntervalMinutes(),
		"md_content":       hb.MDContent(),
		"agent_code":       hb.AgentCode(),
		"requirement_type": hb.RequirementType(),
		"sort_order":       hb.SortOrder(),
		"created_at":       hb.CreatedAt().UnixMilli(),
		"updated_at":       hb.UpdatedAt().UnixMilli(),
	}
}
