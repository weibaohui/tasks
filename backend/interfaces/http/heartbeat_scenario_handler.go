package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type HeartbeatScenarioHandler struct {
	scenarioService *application.HeartbeatScenarioService
}

func NewHeartbeatScenarioHandler(scenarioService *application.HeartbeatScenarioService) *HeartbeatScenarioHandler {
	return &HeartbeatScenarioHandler{scenarioService: scenarioService}
}

func (h *HeartbeatScenarioHandler) CreateScenario(c *gin.Context) {
	var req struct {
		Code        string                         `json:"code" binding:"required"`
		Name        string                         `json:"name" binding:"required"`
		Description string                         `json:"description"`
		Items       []domain.HeartbeatScenarioItem `json:"items"`
		Enabled     bool                           `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	scenario, err := h.scenarioService.CreateScenario(c.Request.Context(), req.Code, req.Name, req.Description, req.Items, req.Enabled)
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, heartbeatScenarioToMap(scenario))
}

func (h *HeartbeatScenarioHandler) ListScenarios(c *gin.Context) {
	scenarios, err := h.scenarioService.ListScenarios(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(scenarios))
	for _, s := range scenarios {
		resp = append(resp, heartbeatScenarioToMap(s))
	}
	c.JSON(http.StatusOK, resp)
}

func (h *HeartbeatScenarioHandler) GetScenario(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "code is required"})
		return
	}
	scenario, err := h.scenarioService.GetScenarioByCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	if scenario == nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "scenario not found"})
		return
	}
	c.JSON(http.StatusOK, heartbeatScenarioToMap(scenario))
}

func (h *HeartbeatScenarioHandler) UpdateScenario(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "code is required"})
		return
	}
	var req struct {
		Name        string                         `json:"name" binding:"required"`
		Description string                         `json:"description"`
		Items       []domain.HeartbeatScenarioItem `json:"items"`
		Enabled     bool                           `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	scenario, err := h.scenarioService.UpdateScenario(c.Request.Context(), code, req.Name, req.Description, req.Items, req.Enabled)
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, heartbeatScenarioToMap(scenario))
}

func (h *HeartbeatScenarioHandler) DeleteScenario(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	if err := h.scenarioService.DeleteScenario(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"message": "ok"})
}

func (h *HeartbeatScenarioHandler) ApplyScenarioToProject(c *gin.Context) {
	projectID := c.Param("project_id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "project_id is required"})
		return
	}
	var req struct {
		ScenarioCode string `json:"scenario_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	if req.ScenarioCode == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "scenario_code is required"})
		return
	}

	// 从上下文中获取 projectService（由路由注入）
	projectService, exists := c.Get("projectService")
	if !exists {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "project service not available"})
		return
	}
	svc, ok := projectService.(*application.ProjectApplicationService)
	if !ok {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "project service invalid"})
		return
	}

	if err := svc.ApplyScenarioToProject(c.Request.Context(), projectID, req.ScenarioCode); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"message": "ok"})
}

func heartbeatScenarioToMap(scenario *domain.HeartbeatScenario) map[string]interface{} {
	items := scenario.Items()
	itemMaps := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		itemMaps = append(itemMaps, map[string]interface{}{
			"name":             item.Name,
			"interval_minutes": item.IntervalMinutes,
			"md_content":       item.MDContent,
			"agent_code":       item.AgentCode,
			"requirement_type": item.RequirementType,
			"sort_order":       item.SortOrder,
		})
	}
	return map[string]interface{}{
		"id":          scenario.ID().String(),
		"code":        scenario.Code(),
		"name":        scenario.Name(),
		"description": scenario.Description(),
		"items":       itemMaps,
		"enabled":     scenario.Enabled(),
		"is_built_in": scenario.IsBuiltIn(),
		"created_at":  scenario.CreatedAt().UnixMilli(),
		"updated_at":  scenario.UpdatedAt().UnixMilli(),
	}
}
