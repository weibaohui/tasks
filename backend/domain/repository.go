/**
 * 仓储接口定义
 * 定义数据访问契约，由基础设施层实现
 */
package domain

import (
	"context"
	"time"
)

// TaskRepository 任务仓储接口
type TaskRepository interface {
	// Save 保存任务
	Save(ctx context.Context, task *Task) error
	// FindByID 根据ID查找任务
	FindByID(ctx context.Context, id TaskID) (*Task, error)
	// FindAll 获取所有任务
	FindAll(ctx context.Context) ([]*Task, error)
	// FindByTraceID 根据TraceID查找所有任务
	FindByTraceID(ctx context.Context, traceID TraceID) ([]*Task, error)
	// FindByParentID 根据父任务ID查找子任务
	FindByParentID(ctx context.Context, parentID TaskID) ([]*Task, error)
	// FindByStatus 根据状态查找任务
	FindByStatus(ctx context.Context, status TaskStatus) ([]*Task, error)
	// FindRunningTasks 查找所有运行中的任务
	FindRunningTasks(ctx context.Context) ([]*Task, error)
	// Delete 删除任务
	Delete(ctx context.Context, id TaskID) error
	// Exists 判断任务是否存在
	Exists(ctx context.Context, id TaskID) (bool, error)
}

type UserRepository interface {
	Save(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id UserID) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	FindByUserCode(ctx context.Context, userCode UserCode) (*User, error)
	FindAll(ctx context.Context) ([]*User, error)
	Delete(ctx context.Context, id UserID) error
}

type AgentRepository interface {
	Save(ctx context.Context, agent *Agent) error
	FindByID(ctx context.Context, id AgentID) (*Agent, error)
	FindByAgentCode(ctx context.Context, code AgentCode) (*Agent, error)
	FindByUserCode(ctx context.Context, userCode string) ([]*Agent, error)
	FindAll(ctx context.Context) ([]*Agent, error)
	Delete(ctx context.Context, id AgentID) error
}

type LLMProviderRepository interface {
	Save(ctx context.Context, provider *LLMProvider) error
	FindByID(ctx context.Context, id LLMProviderID) (*LLMProvider, error)
	FindByUserCode(ctx context.Context, userCode string) ([]*LLMProvider, error)
	FindDefaultActive(ctx context.Context, userCode string) (*LLMProvider, error)
	ClearDefaultByUserCode(ctx context.Context, userCode string, excludeID *LLMProviderID) error
	Delete(ctx context.Context, id LLMProviderID) error
}

type ChannelRepository interface {
	Save(ctx context.Context, channel *Channel) error
	FindByID(ctx context.Context, id ChannelID) (*Channel, error)
	FindByCode(ctx context.Context, code ChannelCode) (*Channel, error)
	FindByUserCode(ctx context.Context, userCode string) ([]*Channel, error)
	FindByAgentCode(ctx context.Context, agentCode string) ([]*Channel, error)
	FindActiveByUserCode(ctx context.Context, userCode string) ([]*Channel, error)
	FindActive(ctx context.Context) ([]*Channel, error)
	Delete(ctx context.Context, id ChannelID) error
}

type SessionRepository interface {
	Save(ctx context.Context, session *Session) error
	FindByID(ctx context.Context, id SessionID) (*Session, error)
	FindBySessionKey(ctx context.Context, sessionKey string) (*Session, error)
	FindByUserCode(ctx context.Context, userCode string) ([]*Session, error)
	FindByChannelCode(ctx context.Context, channelCode string) ([]*Session, error)
	FindActiveByUserCode(ctx context.Context, userCode string) ([]*Session, error)
	DeleteBySessionKey(ctx context.Context, sessionKey string) error
	DeleteByChannelCode(ctx context.Context, channelCode string) error
}

type ConversationRecordListFilter struct {
	TraceID     string
	SessionKey  string
	UserCode    string
	AgentCode   string
	ChannelCode string
	EventType   string
	Role        string
	Limit       int
	Offset      int
}

type ConversationStatsFilter struct {
	StartTime    *time.Time
	EndTime      *time.Time
	AgentCodes   []string
	ChannelCodes []string
	Roles        []string
}

type DailyTokenTrend struct {
	Date             string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type ConversationStats struct {
	TotalPromptTokens     int
	TotalCompletionTokens int
	TotalTokens           int
	DailyTrends           []DailyTokenTrend
	AgentDistribution     []AgentStats
	ChannelDistribution   []ChannelStats
	RoleDistribution      []RoleStats
	TotalSessions         int
	TotalRecords          int
}

type AgentStats struct {
	Code   string
	Name   string
	Count  int
	Tokens int
}

type ChannelStats struct {
	Type  string
	Count int
}

type RoleStats struct {
	Role  string
	Count int
}

type ConversationRecordRepository interface {
	Save(ctx context.Context, record *ConversationRecord) error
	FindByID(ctx context.Context, id ConversationRecordID) (*ConversationRecord, error)
	FindByTraceID(ctx context.Context, traceID string, limit int) ([]*ConversationRecord, error)
	FindBySessionKey(ctx context.Context, sessionKey string, limit int) ([]*ConversationRecord, error)
	List(ctx context.Context, filter ConversationRecordListFilter) ([]*ConversationRecord, error)
	GetStats(ctx context.Context, filter ConversationStatsFilter) (*ConversationStats, error)
}

// EventStore 事件存储接口
type EventStore interface {
	// Save 保存事件
	Save(ctx context.Context, event DomainEvent) error
	// FindByTraceID 根据TraceID查找所有事件
	FindByTraceID(ctx context.Context, traceID TraceID) ([]DomainEvent, error)
}

// IDGenerator ID生成器接口
type IDGenerator interface {
	// Generate 生成ID
	Generate() string
}

// MCP 相关仓储
type MCPServerRepository interface {
	Create(ctx context.Context, server *MCPServer) error
	Update(ctx context.Context, server *MCPServer) error
	Delete(ctx context.Context, id MCPServerID) error
	GetByID(ctx context.Context, id MCPServerID) (*MCPServer, error)
	GetByCode(ctx context.Context, code string) (*MCPServer, error)
	List(ctx context.Context) ([]*MCPServer, error)
	ListByStatus(ctx context.Context, status string) ([]*MCPServer, error)
	CheckCodeExists(ctx context.Context, code string) (bool, error)
}

type AgentMCPBindingRepository interface {
	Create(ctx context.Context, binding *AgentMCPBinding) error
	Update(ctx context.Context, binding *AgentMCPBinding) error
	Delete(ctx context.Context, id AgentMCPBindingID) error
	DeleteByAgentAndMCPServer(ctx context.Context, agentID AgentID, serverID MCPServerID) error
	GetByID(ctx context.Context, id AgentMCPBindingID) (*AgentMCPBinding, error)
	GetByAgentID(ctx context.Context, agentID AgentID) ([]*AgentMCPBinding, error)
	CheckExists(ctx context.Context, agentID AgentID, serverID MCPServerID) (bool, error)
}

type MCPToolModel struct {
	ID          string     `json:"id"`
	MCPServerID MCPServerID `json:"mcp_server_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	InputSchema string     `json:"input_schema"` // JSON
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type MCPToolRepository interface {
	Create(ctx context.Context, tool *MCPToolModel) error
	DeleteByServerID(ctx context.Context, serverID MCPServerID) error
	ListByServerID(ctx context.Context, serverID MCPServerID) ([]*MCPToolModel, error)
}

type MCPToolLog struct {
	ID          string
	SessionKey  string
	MCPServerID MCPServerID
	ToolName    string
	Parameters  string // JSON
	Result      string
	ErrorMsg    string
	ExecuteTime uint
	CreatedAt   time.Time
}

type MCPToolLogRepository interface {
	Create(ctx context.Context, log *MCPToolLog) error
	ListByServerID(ctx context.Context, serverID MCPServerID, limit int) ([]*MCPToolLog, error)
}
