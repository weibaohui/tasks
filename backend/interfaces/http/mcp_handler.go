package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type MCPHandler struct {
	service *application.MCPApplicationService
}

// NewMCPHandler 构造 MCP HTTP 处理器
func NewMCPHandler(service *application.MCPApplicationService) *MCPHandler {
	return &MCPHandler{service: service}
}

// 请求与响应结构
type CreateServerRequest struct {
	Code          string                  `json:"code"`
	Name          string                  `json:"name"`
	Description   string                  `json:"description"`
	TransportType domain.MCPTransportType `json:"transport_type"`
	Command       string                  `json:"command"`
	Args          []string                `json:"args"`
	URL           string                  `json:"url"`
	EnvVars       map[string]string       `json:"env_vars"`
}

type UpdateServerRequest struct {
	Name          *string                  `json:"name"`
	Description   *string                  `json:"description"`
	TransportType *domain.MCPTransportType `json:"transport_type"`
	Command       *string                  `json:"command"`
	Args          *[]string                `json:"args"`
	URL           *string                  `json:"url"`
	EnvVars       *map[string]string       `json:"env_vars"`
}

// CreateServer 创建 MCP 服务器
func (h *MCPHandler) CreateServer(c *gin.Context) {
	var req CreateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "请求格式错误"})
		return
	}
	server, err := h.service.CreateServer(c.Request.Context(), application.CreateMCPServerCommand{
		Code:          req.Code,
		Name:          req.Name,
		Description:   req.Description,
		TransportType: req.TransportType,
		Command:       req.Command,
		Args:          req.Args,
		URL:           req.URL,
		EnvVars:       req.EnvVars,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, serverToMap(server))
}

// ListServers 列出 MCP 服务器
func (h *MCPHandler) ListServers(c *gin.Context) {
	servers, err := h.service.ListServers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(servers))
	for _, s := range servers {
		resp = append(resp, serverToMap(s))
	}
	c.JSON(http.StatusOK, resp)
}

// GetServer 获取 MCP 服务器
func (h *MCPHandler) GetServer(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id 必填"})
		return
	}
	server, err := h.service.GetServer(c.Request.Context(), domain.NewMCPServerID(id))
	if err != nil || server == nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "未找到服务器"})
		return
	}
	c.JSON(http.StatusOK, serverToMap(server))
}

// UpdateServer 更新 MCP 服务器
func (h *MCPHandler) UpdateServer(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id 必填"})
		return
	}
	var req UpdateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "请求格式错误"})
		return
	}
	server, err := h.service.UpdateServer(c.Request.Context(), application.UpdateMCPServerCommand{
		ID:            domain.NewMCPServerID(id),
		Name:          req.Name,
		Description:   req.Description,
		TransportType: req.TransportType,
		Command:       req.Command,
		Args:          req.Args,
		URL:           req.URL,
		EnvVars:       req.EnvVars,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, serverToMap(server))
}

// DeleteServer 删除 MCP 服务器
func (h *MCPHandler) DeleteServer(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id 必填"})
		return
	}
	if err := h.service.DeleteServer(c.Request.Context(), domain.NewMCPServerID(id)); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"message": "ok"})
}

// TestServer 测试 MCP 服务器连接
func (h *MCPHandler) TestServer(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id 必填"})
		return
	}
	if err := h.service.TestServer(c.Request.Context(), domain.NewMCPServerID(id)); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"message": "ok"})
}

// RefreshCapabilities 刷新 MCP 工具能力
func (h *MCPHandler) RefreshCapabilities(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id 必填"})
		return
	}
	if err := h.service.RefreshCapabilities(c.Request.Context(), domain.NewMCPServerID(id)); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"message": "ok"})
}

// ListTools 列出 MCP 工具
func (h *MCPHandler) ListTools(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id 必填"})
		return
	}
	tools, err := h.service.ListTools(c.Request.Context(), domain.NewMCPServerID(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, tools)
}

// handleGetServers 根据 query 参数分发到 GetServer 或 ListServers
func (h *MCPHandler) handleGetServers(c *gin.Context) {
	if c.Query("id") != "" {
		h.GetServer(c)
		return
	}
	h.ListServers(c)
}

// 绑定
type CreateBindingRequest struct {
	AgentID      string   `json:"agent_id"`
	MCPServerID  string   `json:"mcp_server_id"`
	EnabledTools []string `json:"enabled_tools"`
	IsActive     *bool    `json:"is_active"`
	AutoLoad     *bool    `json:"auto_load"`
}

type UpdateBindingRequest struct {
	EnabledTools *[]string `json:"enabled_tools"`
	IsActive     *bool     `json:"is_active"`
	AutoLoad     *bool     `json:"auto_load"`
}

// ListBindings 列出 Agent-MCP 绑定
func (h *MCPHandler) ListBindings(c *gin.Context) {
	agentID := c.Query("agent_id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "agent_id 必填"})
		return
	}
	list, err := h.service.ListAgentBindings(c.Request.Context(), domain.NewAgentID(agentID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(list))
	for _, b := range list {
		resp = append(resp, bindingToMap(b))
	}
	c.JSON(http.StatusOK, resp)
}

// CreateBinding 创建 Agent-MCP 绑定
func (h *MCPHandler) CreateBinding(c *gin.Context) {
	var req CreateBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "请求格式错误"})
		return
	}
	binding, err := h.service.CreateAgentBinding(c.Request.Context(), application.CreateAgentMCPBindingCommand{
		AgentID:      domain.NewAgentID(req.AgentID),
		MCPServerID:  domain.NewMCPServerID(req.MCPServerID),
		EnabledTools: req.EnabledTools,
		IsActive:     req.IsActive,
		AutoLoad:     req.AutoLoad,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, bindingToMap(binding))
}

// UpdateBinding 更新 Agent-MCP 绑定
func (h *MCPHandler) UpdateBinding(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id 必填"})
		return
	}
	var req UpdateBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "请求格式错误"})
		return
	}
	binding, err := h.service.UpdateAgentBinding(c.Request.Context(), application.UpdateAgentMCPBindingCommand{
		ID:           domain.NewAgentMCPBindingID(id),
		EnabledTools: req.EnabledTools,
		IsActive:     req.IsActive,
		AutoLoad:     req.AutoLoad,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, bindingToMap(binding))
}

// DeleteBinding 删除 Agent-MCP 绑定
func (h *MCPHandler) DeleteBinding(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id 必填"})
		return
	}
	if err := h.service.DeleteAgentBinding(c.Request.Context(), domain.NewAgentMCPBindingID(id)); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"message": "ok"})
}

func serverToMap(s *domain.MCPServer) map[string]interface{} {
	return map[string]interface{}{
		"id":             s.ID().String(),
		"code":           s.Code(),
		"name":           s.Name(),
		"description":    s.Description(),
		"transport_type": s.TransportType(),
		"command":        s.Command(),
		"args":           s.Args(),
		"url":            s.URL(),
		"env_vars":       s.EnvVars(),
		"status":         s.Status(),
		"capabilities":   s.Capabilities(),
		"last_connected": unixMilliPtr(s.LastConnectedAt()),
		"error_message":  s.ErrorMessage(),
		"created_at":     s.CreatedAt().UnixMilli(),
		"updated_at":     s.UpdatedAt().UnixMilli(),
	}
}

func bindingToMap(b *domain.AgentMCPBinding) map[string]interface{} {
	return map[string]interface{}{
		"id":            b.ID().String(),
		"agent_id":      b.AgentID().String(),
		"mcp_server_id": b.MCPServerID().String(),
		"enabled_tools": b.EnabledTools(),
		"is_active":     b.IsActive(),
		"auto_load":     b.AutoLoad(),
		"created_at":    b.CreatedAt().UnixMilli(),
		"updated_at":    b.UpdatedAt().UnixMilli(),
	}
}

func unixMilliPtr(t *time.Time) *int64 {
	if t == nil {
		return nil
	}
	v := t.UnixMilli()
	return &v
}
