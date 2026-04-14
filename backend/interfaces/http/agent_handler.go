package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type AgentHandler struct {
	agentService *application.AgentApplicationService
}

func NewAgentHandler(agentService *application.AgentApplicationService) *AgentHandler {
	return &AgentHandler{agentService: agentService}
}

type CreateAgentRequest struct {
	UserCode              string   `json:"user_code"`
	Name                  string   `json:"name"`
	AgentType             string   `json:"agent_type"`
	Description           string   `json:"description"`
	IdentityContent       string   `json:"identity_content"`
	SoulContent           string   `json:"soul_content"`
	AgentsContent         string   `json:"agents_content"`
	UserContent           string   `json:"user_content"`
	ToolsContent          string   `json:"tools_content"`
	Model                 string   `json:"model"`
	LLMProviderID         *string  `json:"llm_provider_id"`
	MaxTokens             int      `json:"max_tokens"`
	Temperature           float64  `json:"temperature"`
	MaxIterations         int      `json:"max_iterations"`
	HistoryMessages       int      `json:"history_messages"`
	SkillsList            []string `json:"skills_list"`
	ToolsList             []string `json:"tools_list"`
	IsActive              *bool    `json:"is_active"`
	IsDefault             bool     `json:"is_default"`
	EnableThinkingProcess bool     `json:"enable_thinking_process"`
}

type UpdateAgentRequest struct {
	Name                  *string   `json:"name"`
	AgentType             *string   `json:"agent_type"`
	Description           *string   `json:"description"`
	IdentityContent       *string   `json:"identity_content"`
	SoulContent           *string   `json:"soul_content"`
	AgentsContent         *string   `json:"agents_content"`
	UserContent           *string   `json:"user_content"`
	ToolsContent          *string   `json:"tools_content"`
	Model                 *string   `json:"model"`
	LLMProviderID         *string   `json:"llm_provider_id"`
	MaxTokens             *int      `json:"max_tokens"`
	Temperature           *float64  `json:"temperature"`
	MaxIterations         *int      `json:"max_iterations"`
	HistoryMessages       *int      `json:"history_messages"`
	SkillsList            *[]string `json:"skills_list"`
	ToolsList             *[]string `json:"tools_list"`
	IsActive              *bool     `json:"is_active"`
	IsDefault             *bool     `json:"is_default"`
	EnableThinkingProcess *bool     `json:"enable_thinking_process"`
}

func (h *AgentHandler) CreateAgent(c *gin.Context) {
	var req CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	agent, err := h.agentService.CreateAgent(c.Request.Context(), application.CreateAgentCommand{
		UserCode:              req.UserCode,
		Name:                  req.Name,
		AgentType:             req.AgentType,
		Description:           req.Description,
		IdentityContent:       req.IdentityContent,
		SoulContent:           req.SoulContent,
		AgentsContent:         req.AgentsContent,
		UserContent:           req.UserContent,
		ToolsContent:          req.ToolsContent,
		Model:                 req.Model,
		LLMProviderID:         req.LLMProviderID,
		MaxTokens:             req.MaxTokens,
		Temperature:           req.Temperature,
		MaxIterations:         req.MaxIterations,
		HistoryMessages:       req.HistoryMessages,
		SkillsList:            req.SkillsList,
		ToolsList:             req.ToolsList,
		IsActive:              req.IsActive,
		IsDefault:             req.IsDefault,
		EnableThinkingProcess: req.EnableThinkingProcess,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, agentToMap(agent))
}

func (h *AgentHandler) ListAgents(c *gin.Context) {
	userCode := c.Query("user_code")
	agents, err := h.agentService.ListAgents(c.Request.Context(), userCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(agents))
	for _, agent := range agents {
		resp = append(resp, agentToMap(agent))
	}
	c.JSON(http.StatusOK, resp)
}

func (h *AgentHandler) GetAgent(c *gin.Context) {
	id := c.Query("id")
	code := c.Query("code")
	if id == "" && code == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id or code is required"})
		return
	}

	var (
		agent *domain.Agent
		err   error
	)
	if id != "" {
		agent, err = h.agentService.GetAgent(c.Request.Context(), domain.NewAgentID(id))
	} else {
		agent, err = h.agentService.GetAgentByCode(c.Request.Context(), domain.NewAgentCode(code))
	}
	if err != nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, agentToMap(agent))
}

func (h *AgentHandler) UpdateAgent(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	var req UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	agent, err := h.agentService.UpdateAgent(c.Request.Context(), application.UpdateAgentCommand{
		ID:                    domain.NewAgentID(id),
		Name:                  req.Name,
		AgentType:             req.AgentType,
		Description:           req.Description,
		IdentityContent:       req.IdentityContent,
		SoulContent:           req.SoulContent,
		AgentsContent:         req.AgentsContent,
		UserContent:           req.UserContent,
		ToolsContent:          req.ToolsContent,
		Model:                 req.Model,
		LLMProviderID:         req.LLMProviderID,
		MaxTokens:             req.MaxTokens,
		Temperature:           req.Temperature,
		MaxIterations:         req.MaxIterations,
		HistoryMessages:       req.HistoryMessages,
		SkillsList:            req.SkillsList,
		ToolsList:             req.ToolsList,
		IsActive:              req.IsActive,
		IsDefault:             req.IsDefault,
		EnableThinkingProcess: req.EnableThinkingProcess,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, agentToMap(agent))
}

// PatchAgentRequest 局部更新请求，指针字段区分"未提供"与"零值"
type PatchAgentRequest struct {
	Name                  *string                  `json:"name"`
	AgentType             *string                  `json:"agent_type"`
	Description           *string                  `json:"description"`
	IdentityContent       *string                  `json:"identity_content"`
	SoulContent           *string                  `json:"soul_content"`
	AgentsContent         *string                  `json:"agents_content"`
	UserContent           *string                  `json:"user_content"`
	ToolsContent          *string                  `json:"tools_content"`
	Model                 *string                  `json:"model"`
	LLMProviderID         *string                  `json:"llm_provider_id"`
	MaxTokens             *int                     `json:"max_tokens"`
	Temperature           *float64                 `json:"temperature"`
	MaxIterations         *int                     `json:"max_iterations"`
	HistoryMessages       *int                     `json:"history_messages"`
	SkillsList            *[]string                `json:"skills_list"`
	ToolsList             *[]string                `json:"tools_list"`
	IsActive              *bool                    `json:"is_active"`
	IsDefault             *bool                    `json:"is_default"`
	EnableThinkingProcess *bool                    `json:"enable_thinking_process"`
	ClaudeCodeConfig      *domain.ClaudeCodeConfig `json:"claude_code_config"`
	OpenCodeConfig        *domain.OpenCodeConfig   `json:"opencode_config"`
}

func (h *AgentHandler) PatchAgent(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	var req PatchAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}

	agent, err := h.agentService.PatchAgent(c.Request.Context(), application.PatchAgentCommand{
		ID:                    domain.NewAgentID(id),
		Name:                  req.Name,
		AgentType:             req.AgentType,
		Description:           req.Description,
		IdentityContent:       req.IdentityContent,
		SoulContent:           req.SoulContent,
		AgentsContent:         req.AgentsContent,
		UserContent:           req.UserContent,
		ToolsContent:          req.ToolsContent,
		Model:                 req.Model,
		LLMProviderID:         req.LLMProviderID,
		MaxTokens:             req.MaxTokens,
		Temperature:           req.Temperature,
		MaxIterations:         req.MaxIterations,
		HistoryMessages:       req.HistoryMessages,
		SkillsList:            req.SkillsList,
		ToolsList:             req.ToolsList,
		IsActive:              req.IsActive,
		IsDefault:             req.IsDefault,
		EnableThinkingProcess: req.EnableThinkingProcess,
		ClaudeCodeConfig:      req.ClaudeCodeConfig,
		OpenCodeConfig:        req.OpenCodeConfig,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, agentToMap(agent))
}

func (h *AgentHandler) DeleteAgent(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	if err := h.agentService.DeleteAgent(c.Request.Context(), domain.NewAgentID(id)); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"message": "ok"})
}

// handleGetAgents 根据 query 参数分发到 GetAgent 或 ListAgents
func (h *AgentHandler) handleGetAgents(c *gin.Context) {
	if c.Query("id") != "" || c.Query("code") != "" {
		h.GetAgent(c)
		return
	}
	h.ListAgents(c)
}

func agentToMap(agent *domain.Agent) map[string]interface{} {
	return map[string]interface{}{
		"id":                      agent.ID().String(),
		"agent_code":              agent.AgentCode().String(),
		"agent_type":              agent.AgentType().String(),
		"user_code":               agent.UserCode(),
		"name":                    agent.Name(),
		"description":             agent.Description(),
		"identity_content":        agent.IdentityContent(),
		"soul_content":            agent.SoulContent(),
		"agents_content":          agent.AgentsContent(),
		"user_content":            agent.UserContent(),
		"tools_content":           agent.ToolsContent(),
		"model":                   agent.Model(),
		"llm_provider_id":         agent.LLMProviderID().String(),
		"max_tokens":              agent.MaxTokens(),
		"temperature":             agent.Temperature(),
		"max_iterations":          agent.MaxIterations(),
		"history_messages":        agent.HistoryMessages(),
		"skills_list":             agent.SkillsList(),
		"tools_list":              agent.ToolsList(),
		"is_active":               agent.IsActive(),
		"is_default":              agent.IsDefault(),
		"enable_thinking_process": agent.EnableThinkingProcess(),
		"shadow_from":             agent.ShadowFrom(),
		"claude_code_config":      agent.ClaudeCodeConfig(),
		"opencode_config":         agent.OpenCodeConfig(),
		"created_at":              agent.CreatedAt().UnixMilli(),
		"updated_at":              agent.UpdatedAt().UnixMilli(),
	}
}
