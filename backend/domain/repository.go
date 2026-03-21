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
