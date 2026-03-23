/**
 * 仓储接口定义
 * 定义数据访问契约，由基础设施层实现
 */
package domain

import "context"

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
	Delete(ctx context.Context, id ChannelID) error
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
