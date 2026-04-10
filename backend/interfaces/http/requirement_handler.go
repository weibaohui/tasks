package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type RequirementHandler struct {
	requirementService *application.RequirementApplicationService
	dispatchService    *application.RequirementDispatchService
}

func NewRequirementHandler(requirementService *application.RequirementApplicationService, dispatchService *application.RequirementDispatchService) *RequirementHandler {
	return &RequirementHandler{
		requirementService: requirementService,
		dispatchService:    dispatchService,
	}
}

type CreateRequirementRequest struct {
	ProjectID          string `json:"project_id"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
	TempWorkspaceRoot  string `json:"temp_workspace_root"`
	RequirementType    string `json:"requirement_type"`
}

type UpdateRequirementRequest struct {
	ID                 string  `json:"id"`
	Title              *string `json:"title"`
	Description        *string `json:"description"`
	AcceptanceCriteria *string `json:"acceptance_criteria"`
	TempWorkspaceRoot  *string `json:"temp_workspace_root"`
	RequirementType    *string `json:"requirement_type"`
}

type UpdateRequirementStatusRequest struct {
	ID        string `json:"id"`
	NewStatus string `json:"new_status"`
}

type DispatchRequirementRequest struct {
	RequirementID string `json:"requirement_id"`
	AgentCode     string `json:"agent_code"`
	ChannelCode   string `json:"channel_code"`
	SessionKey    string `json:"session_key"`
}

type ReportRequirementPRRequest struct {
	RequirementID string `json:"requirement_id"`
}

func (h *RequirementHandler) CreateRequirement(c *gin.Context) {
	var req CreateRequirementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	requirement, err := h.requirementService.CreateRequirement(c.Request.Context(), application.CreateRequirementCommand{
		ProjectID:          domain.NewProjectID(req.ProjectID),
		Title:              req.Title,
		Description:        req.Description,
		AcceptanceCriteria: req.AcceptanceCriteria,
		TempWorkspaceRoot:  req.TempWorkspaceRoot,
		RequirementType:    req.RequirementType,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, h.requirementToMap(requirement))
}

func (h *RequirementHandler) GetRequirement(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	requirement, err := h.requirementService.GetRequirement(c.Request.Context(), domain.NewRequirementID(id))
	if err != nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.requirementToMap(requirement))
}

func (h *RequirementHandler) ListRequirements(c *gin.Context) {
	projectIDStr := c.Query("project_id")
	status := c.Query("status")
	limitStr := c.Query("limit")
	offsetStr := c.Query("offset")

	var projectID *domain.ProjectID
	if projectIDStr != "" {
		id := domain.NewProjectID(projectIDStr)
		projectID = &id
	}

	// 分页模式（看板视图使用）
	if limitStr != "" || offsetStr != "" {
		limit, _ := strconv.Atoi(limitStr)
		offset, _ := strconv.Atoi(offsetStr)
		if limit <= 0 {
			limit = 10
		}
		var statuses []string
		if status != "" {
			statuses = strings.Split(status, ",")
		}
		requirements, total, err := h.requirementService.ListRequirementsPaginated(
			c.Request.Context(),
			application.ListRequirementsQuery{
				ProjectID: projectID,
				Statuses:  statuses,
				Limit:     limit,
				Offset:    offset,
			},
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
			return
		}
		items := make([]map[string]interface{}, 0, len(requirements))
		for _, req := range requirements {
			items = append(items, h.requirementToMap(req))
		}
		c.JSON(http.StatusOK, map[string]interface{}{
			"items": items,
			"total": total,
		})
		return
	}

	// 兼容模式：返回全部数据（表格视图使用）
	requirements, err := h.requirementService.ListRequirements(c.Request.Context(), projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(requirements))
	for _, requirement := range requirements {
		resp = append(resp, h.requirementToMap(requirement))
	}
	c.JSON(http.StatusOK, resp)
}

func (h *RequirementHandler) UpdateRequirement(c *gin.Context) {
	var req UpdateRequirementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	requirement, err := h.requirementService.UpdateRequirement(c.Request.Context(), application.UpdateRequirementCommand{
		ID:                 domain.NewRequirementID(req.ID),
		Title:              req.Title,
		Description:        req.Description,
		AcceptanceCriteria: req.AcceptanceCriteria,
		TempWorkspaceRoot:  req.TempWorkspaceRoot,
		RequirementType:    req.RequirementType,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.requirementToMap(requirement))
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

// GetStatusStats 获取状态统计数据（动态从数据库提取）
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

func (h *RequirementHandler) DispatchRequirement(c *gin.Context) {
	var req DispatchRequirementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	result, err := h.dispatchService.DispatchRequirement(c.Request.Context(), application.DispatchRequirementCommand{
		RequirementID: domain.NewRequirementID(req.RequirementID),
		AgentCode:     req.AgentCode,
		ChannelCode:   req.ChannelCode,
		SessionKey:    req.SessionKey,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *RequirementHandler) ReportRequirementPROpened(c *gin.Context) {
	var req ReportRequirementPRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	requirement, err := h.requirementService.ReportRequirementPROpened(c.Request.Context(), application.ReportRequirementPRCommand{
		ID: domain.NewRequirementID(req.RequirementID),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.requirementToMap(requirement))
}

type RedispatchRequirementRequest struct {
	RequirementID string `json:"requirement_id"`
}

type DeleteRequirementRequest struct {
	ID string `json:"id"`
}

type BatchDeleteRequirementsRequest struct {
	IDs []string `json:"ids"`
}

func (h *RequirementHandler) RedispatchRequirement(c *gin.Context) {
	var req RedispatchRequirementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	requirement, err := h.requirementService.RedispatchRequirement(c.Request.Context(), application.RedispatchRequirementCommand{
		ID: domain.NewRequirementID(req.RequirementID),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.requirementToMap(requirement))
}

// CopyAndDispatchRequirement 复制需求并派发新副本
func (h *RequirementHandler) CopyAndDispatchRequirement(c *gin.Context) {
	var req RedispatchRequirementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	requirement, err := h.requirementService.CopyAndDispatchRequirement(c.Request.Context(), application.CopyAndDispatchRequirementCommand{
		ID: domain.NewRequirementID(req.RequirementID),
	}, h.dispatchService)
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.requirementToMap(requirement))
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

func (h *RequirementHandler) requirementToMap(requirement *domain.Requirement) map[string]interface{} {
	startedAt := interface{}(nil)
	if requirement.StartedAt() != nil {
		startedAt = requirement.StartedAt().UnixMilli()
	}
	completedAt := interface{}(nil)
	if requirement.CompletedAt() != nil {
		completedAt = requirement.CompletedAt().UnixMilli()
	}
	resp := map[string]interface{}{
		"id":                   requirement.ID().String(),
		"project_id":           requirement.ProjectID().String(),
		"title":                requirement.Title(),
		"description":          requirement.Description(),
		"acceptance_criteria":  requirement.AcceptanceCriteria(),
		"temp_workspace_root":  requirement.TempWorkspaceRoot(),
		"status":               requirement.Status(),
		"assignee_agent_code":  requirement.AssigneeAgentCode(),
		"replica_agent_code":   requirement.ReplicaAgentCode(),
		"dispatch_session_key": requirement.DispatchSessionKey(),
		"workspace_path":       requirement.WorkspacePath(),
		"last_error":           requirement.LastError(),
		"trace_id":             requirement.TraceID(),
		"prompt_tokens":        requirement.PromptTokens(),
		"completion_tokens":    requirement.CompletionTokens(),
		"total_tokens":         requirement.TotalTokens(),
		"started_at":           startedAt,
		"completed_at":         completedAt,
		"created_at":           requirement.CreatedAt().UnixMilli(),
		"updated_at":           requirement.UpdatedAt().UnixMilli(),
		"requirement_type":     requirement.RequirementType(),
	}
	resp["claude_runtime"] = h.getClaudeRuntimeByRequirement(requirement)
	return resp
}

func (h *RequirementHandler) getClaudeRuntimeByRequirement(requirement *domain.Requirement) map[string]interface{} {
	result := make(map[string]interface{})

	// 先检查 nil，避免空指针
	if requirement == nil {
		return result
	}

	// 从 requirement 直接获取所有 Claude Runtime 状态字段（已持久化到数据库）
	result["prompt"] = requirement.ClaudeRuntimePrompt()
	result["result"] = requirement.ClaudeRuntimeResult()
	result["status"] = requirement.ClaudeRuntimeStatus()
	result["last_error"] = requirement.ClaudeRuntimeError()

	if startedAt := requirement.ClaudeRuntimeStartedAt(); startedAt != nil {
		result["started_at"] = startedAt.UnixMilli()
	} else {
		result["started_at"] = nil
	}

	if endedAt := requirement.ClaudeRuntimeEndedAt(); endedAt != nil {
		result["ended_at"] = endedAt.UnixMilli()
	} else {
		result["ended_at"] = nil
	}

	return result
}

// handleDeleteError 处理删除操作的错误，根据错误类型返回相应的状态码
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

// handleGetRequirements 根据 query 参数分发到 GetRequirement 或 ListRequirements
func (h *RequirementHandler) handleGetRequirements(c *gin.Context) {
	if c.Query("id") != "" {
		h.GetRequirement(c)
		return
	}
	h.ListRequirements(c)
}
