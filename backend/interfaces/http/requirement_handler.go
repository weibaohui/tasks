package http

import (
	"context"
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
	agentRepo          domain.AgentRepository
}

func NewRequirementHandler(requirementService *application.RequirementApplicationService, dispatchService *application.RequirementDispatchService, agentRepo domain.AgentRepository) *RequirementHandler {
	return &RequirementHandler{
		requirementService: requirementService,
		dispatchService:    dispatchService,
		agentRepo:          agentRepo,
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
	resp["assignee_agent"] = h.agentBriefByCode(requirement.AssigneeAgentCode())
	resp["replica_agent"] = h.agentBriefByCode(requirement.ReplicaAgentCode())
	return resp
}

func (h *RequirementHandler) agentBriefByCode(code string) map[string]interface{} {
	if code == "" || h.agentRepo == nil {
		return nil
	}
	agent, err := h.agentRepo.FindByAgentCode(context.Background(), domain.NewAgentCode(code))
	if err != nil || agent == nil {
		return map[string]interface{}{
			"id":         "",
			"agent_code": code,
			"name":       "",
			"shadow_from": "",
		}
	}
	return map[string]interface{}{
		"id":          agent.ID().String(),
		"agent_code":  agent.AgentCode().String(),
		"name":        agent.Name(),
		"shadow_from": agent.ShadowFrom(),
	}
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


// handleGetRequirements 根据 query 参数分发到 GetRequirement 或 ListRequirements
func (h *RequirementHandler) handleGetRequirements(c *gin.Context) {
	if c.Query("id") != "" {
		h.GetRequirement(c)
		return
	}
	h.ListRequirements(c)
}
