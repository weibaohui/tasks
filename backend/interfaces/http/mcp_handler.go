package http

import (
	"encoding/json"
	"net/http"
	"time"

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
func (h *MCPHandler) CreateServer(w http.ResponseWriter, r *http.Request) {
	var req CreateServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "请求格式错误"})
		return
	}
	server, err := h.service.CreateServer(r.Context(), application.CreateMCPServerCommand{
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
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(serverToMap(server))
}

// ListServers 列出 MCP 服务器
func (h *MCPHandler) ListServers(w http.ResponseWriter, r *http.Request) {
	servers, err := h.service.ListServers(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(servers))
	for _, s := range servers {
		resp = append(resp, serverToMap(s))
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// GetServer 获取 MCP 服务器
func (h *MCPHandler) GetServer(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id 必填"})
		return
	}
	server, err := h.service.GetServer(r.Context(), domain.NewMCPServerID(id))
	if err != nil || server == nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusNotFound, Message: "未找到服务器"})
		return
	}
	_ = json.NewEncoder(w).Encode(serverToMap(server))
}

// UpdateServer 更新 MCP 服务器
func (h *MCPHandler) UpdateServer(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id 必填"})
		return
	}
	var req UpdateServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "请求格式错误"})
		return
	}
	server, err := h.service.UpdateServer(r.Context(), application.UpdateMCPServerCommand{
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
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(serverToMap(server))
}

// DeleteServer 删除 MCP 服务器
func (h *MCPHandler) DeleteServer(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id 必填"})
		return
	}
	if err := h.service.DeleteServer(r.Context(), domain.NewMCPServerID(id)); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
}

// TestServer 测试 MCP 服务器连接
func (h *MCPHandler) TestServer(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id 必填"})
		return
	}
	if err := h.service.TestServer(r.Context(), domain.NewMCPServerID(id)); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
}

// RefreshCapabilities 刷新 MCP 工具能力
func (h *MCPHandler) RefreshCapabilities(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id 必填"})
		return
	}
	if err := h.service.RefreshCapabilities(r.Context(), domain.NewMCPServerID(id)); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
}

// ListTools 列出 MCP 工具
func (h *MCPHandler) ListTools(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id 必填"})
		return
	}
	tools, err := h.service.ListTools(r.Context(), domain.NewMCPServerID(id))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(tools)
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
func (h *MCPHandler) ListBindings(w http.ResponseWriter, r *http.Request) {
	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "agent_id 必填"})
		return
	}
	list, err := h.service.ListAgentBindings(r.Context(), domain.NewAgentID(agentID))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(list))
	for _, b := range list {
		resp = append(resp, bindingToMap(b))
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// CreateBinding 创建 Agent-MCP 绑定
func (h *MCPHandler) CreateBinding(w http.ResponseWriter, r *http.Request) {
	var req CreateBindingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "请求格式错误"})
		return
	}
	binding, err := h.service.CreateAgentBinding(r.Context(), application.CreateAgentMCPBindingCommand{
		AgentID:      domain.NewAgentID(req.AgentID),
		MCPServerID:  domain.NewMCPServerID(req.MCPServerID),
		EnabledTools: req.EnabledTools,
		IsActive:     req.IsActive,
		AutoLoad:     req.AutoLoad,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(bindingToMap(binding))
}

// UpdateBinding 更新 Agent-MCP 绑定
func (h *MCPHandler) UpdateBinding(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id 必填"})
		return
	}
	var req UpdateBindingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "请求格式错误"})
		return
	}
	binding, err := h.service.UpdateAgentBinding(r.Context(), application.UpdateAgentMCPBindingCommand{
		ID:           domain.NewAgentMCPBindingID(id),
		EnabledTools: req.EnabledTools,
		IsActive:     req.IsActive,
		AutoLoad:     req.AutoLoad,
	})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(bindingToMap(binding))
}

// DeleteBinding 删除 Agent-MCP 绑定
func (h *MCPHandler) DeleteBinding(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "id 必填"})
		return
	}
	if err := h.service.DeleteAgentBinding(r.Context(), domain.NewAgentMCPBindingID(id)); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
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
