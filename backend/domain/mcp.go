package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrMCPServerIDRequired   = errors.New("mcp server id is required")
	ErrMCPServerCodeRequired = errors.New("mcp server code is required")
	ErrMCPServerNameRequired = errors.New("mcp server name is required")
)

type MCPServerID struct {
	value string
}

func NewMCPServerID(value string) MCPServerID { return MCPServerID{value: value} }
func (id MCPServerID) String() string         { return id.value }

type MCPTransportType string

const (
	MCPTransportSTDIO MCPTransportType = "stdio"
	MCPTransportHTTP  MCPTransportType = "http"
	MCPTransportSSE   MCPTransportType = "sse"
)

type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema,omitempty"`
}

type MCPServer struct {
	id            MCPServerID
	code          string
	name          string
	description   string
	transportType MCPTransportType
	command       string
	args          []string
	url           string
	envVars       map[string]string
	status        string
	capabilities  []MCPTool
	lastConnected *time.Time
	errorMessage  string
	createdAt     time.Time
	updatedAt     time.Time
}

func NewMCPServer(id MCPServerID, code, name string, transport MCPTransportType) (*MCPServer, error) {
	if id.String() == "" {
		return nil, ErrMCPServerIDRequired
	}
	if code == "" {
		return nil, ErrMCPServerCodeRequired
	}
	if name == "" {
		return nil, ErrMCPServerNameRequired
	}
	now := time.Now()
	return &MCPServer{
		id:            id,
		code:          code,
		name:          name,
		transportType: transport,
		status:        "inactive",
		createdAt:     now,
		updatedAt:     now,
	}, nil
}

func (m *MCPServer) ID() MCPServerID                 { return m.id }
func (m *MCPServer) Code() string                    { return m.code }
func (m *MCPServer) Name() string                    { return m.name }
func (m *MCPServer) Description() string             { return m.description }
func (m *MCPServer) TransportType() MCPTransportType { return m.transportType }
func (m *MCPServer) Command() string                 { return m.command }
func (m *MCPServer) Args() []string                  { return append([]string(nil), m.args...) }
func (m *MCPServer) URL() string                     { return m.url }
func (m *MCPServer) EnvVars() map[string]string      { return m.envVars }
func (m *MCPServer) Status() string                  { return m.status }
func (m *MCPServer) Capabilities() []MCPTool         { return append([]MCPTool(nil), m.capabilities...) }
func (m *MCPServer) LastConnectedAt() *time.Time     { return m.lastConnected }
func (m *MCPServer) ErrorMessage() string            { return m.errorMessage }
func (m *MCPServer) CreatedAt() time.Time            { return m.createdAt }
func (m *MCPServer) UpdatedAt() time.Time            { return m.updatedAt }

// MCPProfileUpdate MCP 配置更新参数
type MCPProfileUpdate struct {
	Name        string
	Description string
	Transport   MCPTransportType
	Command     string
	URL         string
	Args        []string
	EnvVars     map[string]string
}

func (m *MCPServer) UpdateProfile(cfg MCPProfileUpdate) {
	if cfg.Name != "" {
		m.name = cfg.Name
	}
	m.description = cfg.Description
	if cfg.Transport != "" {
		m.transportType = cfg.Transport
	}
	if cfg.Command != "" {
		m.command = cfg.Command
	}
	if cfg.URL != "" {
		m.url = cfg.URL
	}
	if cfg.Args != nil {
		m.args = append([]string(nil), cfg.Args...)
	}
	if cfg.EnvVars != nil {
		m.envVars = cfg.EnvVars
	}
	m.updatedAt = time.Now()
}

func (m *MCPServer) SetStatus(status, errorMsg string) {
	m.status = status
	m.errorMessage = errorMsg
	if status == "active" {
		now := time.Now()
		m.lastConnected = &now
	}
	m.updatedAt = time.Now()
}

func (m *MCPServer) SetCapabilities(cap []MCPTool) {
	m.capabilities = append([]MCPTool(nil), cap...)
	m.updatedAt = time.Now()
}

type AgentMCPBindingID struct {
	value string
}

func NewAgentMCPBindingID(value string) AgentMCPBindingID { return AgentMCPBindingID{value: value} }
func (id AgentMCPBindingID) String() string               { return id.value }

type AgentMCPBinding struct {
	id           AgentMCPBindingID
	agentID      AgentID
	mcpServerID  MCPServerID
	enabledTools []string // nil: all enabled
	isActive     bool
	autoLoad     bool
	createdAt    time.Time
	updatedAt    time.Time
}

func NewAgentMCPBinding(id AgentMCPBindingID, agentID AgentID, serverID MCPServerID) *AgentMCPBinding {
	now := time.Now()
	return &AgentMCPBinding{
		id:          id,
		agentID:     agentID,
		mcpServerID: serverID,
		isActive:    true,
		autoLoad:    false,
		createdAt:   now,
		updatedAt:   now,
	}
}

func (b *AgentMCPBinding) ID() AgentMCPBindingID    { return b.id }
func (b *AgentMCPBinding) AgentID() AgentID         { return b.agentID }
func (b *AgentMCPBinding) MCPServerID() MCPServerID { return b.mcpServerID }
func (b *AgentMCPBinding) EnabledTools() []string   { return append([]string(nil), b.enabledTools...) }
func (b *AgentMCPBinding) IsActive() bool           { return b.isActive }
func (b *AgentMCPBinding) AutoLoad() bool           { return b.autoLoad }
func (b *AgentMCPBinding) CreatedAt() time.Time     { return b.createdAt }
func (b *AgentMCPBinding) UpdatedAt() time.Time     { return b.updatedAt }

func (b *AgentMCPBinding) SetEnabledTools(tools []string) {
	if len(tools) == 0 {
		b.enabledTools = nil
	} else {
		b.enabledTools = append([]string(nil), tools...)
	}
	b.updatedAt = time.Now()
}
func (b *AgentMCPBinding) SetActive(active bool) { b.isActive = active; b.updatedAt = time.Now() }
func (b *AgentMCPBinding) SetAutoLoad(auto bool) { b.autoLoad = auto; b.updatedAt = time.Now() }

// snapshots
type MCPServerSnapshot struct {
	ID            MCPServerID
	Code          string
	Name          string
	Description   string
	TransportType MCPTransportType
	Command       string
	Args          []string
	URL           string
	EnvVars       map[string]string
	Status        string
	Capabilities  []MCPTool
	LastConnected *time.Time
	ErrorMessage  string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (m *MCPServer) ToSnapshot() MCPServerSnapshot {
	return MCPServerSnapshot{
		ID:            m.id,
		Code:          m.code,
		Name:          m.name,
		Description:   m.description,
		TransportType: m.transportType,
		Command:       m.command,
		Args:          append([]string(nil), m.args...),
		URL:           m.url,
		EnvVars:       m.envVars,
		Status:        m.status,
		Capabilities:  append([]MCPTool(nil), m.capabilities...),
		LastConnected: m.lastConnected,
		ErrorMessage:  m.errorMessage,
		CreatedAt:     m.createdAt,
		UpdatedAt:     m.updatedAt,
	}
}

func (m *MCPServer) FromSnapshot(s MCPServerSnapshot) {
	m.id = s.ID
	m.code = s.Code
	m.name = s.Name
	m.description = s.Description
	m.transportType = s.TransportType
	m.command = s.Command
	m.args = append([]string(nil), s.Args...)
	m.url = s.URL
	m.envVars = s.EnvVars
	m.status = s.Status
	m.capabilities = append([]MCPTool(nil), s.Capabilities...)
	m.lastConnected = s.LastConnected
	m.errorMessage = s.ErrorMessage
	m.createdAt = s.CreatedAt
	m.updatedAt = s.UpdatedAt
}

type AgentMCPBindingSnapshot struct {
	ID           AgentMCPBindingID
	AgentID      AgentID
	MCPServerID  MCPServerID
	EnabledTools []string
	IsActive     bool
	AutoLoad     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (b *AgentMCPBinding) ToSnapshot() AgentMCPBindingSnapshot {
	return AgentMCPBindingSnapshot{
		ID:           b.id,
		AgentID:      b.agentID,
		MCPServerID:  b.mcpServerID,
		EnabledTools: b.EnabledTools(),
		IsActive:     b.isActive,
		AutoLoad:     b.autoLoad,
		CreatedAt:    b.createdAt,
		UpdatedAt:    b.updatedAt,
	}
}

func (b *AgentMCPBinding) FromSnapshot(s AgentMCPBindingSnapshot) {
	b.id = s.ID
	b.agentID = s.AgentID
	b.mcpServerID = s.MCPServerID
	b.enabledTools = append([]string(nil), s.EnabledTools...)
	b.isActive = s.IsActive
	b.autoLoad = s.AutoLoad
	b.createdAt = s.CreatedAt
	b.updatedAt = s.UpdatedAt
}

// helpers for JSON
func EncodeAny(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}

// MCPToolInfo MCP 工具信息（从 MCP 协议响应映射）
type MCPToolInfo struct {
	Name        string
	Description string
	Schema      map[string]interface{}
}

// MCPToolResult MCP 工具调用结果
type MCPToolResult struct {
	Content string
	IsError bool
}

// MCPClient MCP 客户端接口（屏蔽底层 mcp-go 实现）
type MCPClient interface {
	Start(ctx context.Context) error
	Initialize(ctx context.Context) error
	ListTools(ctx context.Context) ([]MCPToolInfo, error)
	CallTool(ctx context.Context, toolName string, params map[string]interface{}) (MCPToolResult, error)
	Close() error
}

// MCPClientFactory MCP 客户端工厂接口
type MCPClientFactory interface {
	CreateClient(server *MCPServer) (MCPClient, error)
}
