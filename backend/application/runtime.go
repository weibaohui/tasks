/**
 * TaskRuntime 任务运行时管理器
 * 负责管理任务的运行时上下文（context）
 */
package application

import (
	"context"
	"sync"
	"time"
)

// TaskRuntime 任务运行时管理器
type TaskRuntime struct {
	mu       sync.RWMutex
	running  map[string]context.CancelFunc
	contexts map[string]context.Context
}

// NewTaskRuntime 创建任务运行时管理器
func NewTaskRuntime() *TaskRuntime {
	return &TaskRuntime{
		running:  make(map[string]context.CancelFunc),
		contexts: make(map[string]context.Context),
	}
}

// Register 注册任务运行时的 context
func (rt *TaskRuntime) Register(taskID string, ctx context.Context, cancel context.CancelFunc) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.running[taskID] = cancel
	rt.contexts[taskID] = ctx
}

// GetContext 获取任务的 context
func (rt *TaskRuntime) GetContext(taskID string) (context.Context, bool) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	ctx, ok := rt.contexts[taskID]
	return ctx, ok
}

// Cancel 取消任务
func (rt *TaskRuntime) Cancel(taskID string) bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if cancel, ok := rt.running[taskID]; ok {
		cancel()
		delete(rt.running, taskID)
		delete(rt.contexts, taskID)
		return true
	}
	return false
}

// Unregister 注销任务
func (rt *TaskRuntime) Unregister(taskID string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	delete(rt.running, taskID)
	delete(rt.contexts, taskID)
}

// CreateContext 为任务创建带超时的 context
func (rt *TaskRuntime) CreateContext(parentCtx context.Context, taskID string, timeout time.Duration) (context.Context, context.CancelFunc) {
	var ctx context.Context
	var cancel context.CancelFunc

	if timeout > 0 {
		ctx, cancel = context.WithTimeout(parentCtx, timeout)
	} else {
		ctx, cancel = context.WithCancel(parentCtx)
	}

	rt.Register(taskID, ctx, cancel)
	return ctx, cancel
}
