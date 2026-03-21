/**
 * TaskRegistry - 任务注册表
 * 管理所有运行中的任务上下文和 TraceContext
 */
package application

import (
	"sync"
)

type TaskRegistry struct {
	mu            sync.RWMutex
	traceContexts map[string]*TraceContext // traceID -> TraceContext
	taskContexts map[string]*TaskContext // taskID -> TaskContext
	todoLists    map[string]*TodoList    // taskID -> TodoList
}

var (
	defaultRegistry *TaskRegistry
	registryOnce   sync.Once
)

func GetTaskRegistry() *TaskRegistry {
	registryOnce.Do(func() {
		defaultRegistry = &TaskRegistry{
			traceContexts: make(map[string]*TraceContext),
			taskContexts: make(map[string]*TaskContext),
			todoLists:    make(map[string]*TodoList),
		}
	})
	return defaultRegistry
}

func (r *TaskRegistry) RegisterTraceContext(tc *TraceContext) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.traceContexts[tc.TraceID] = tc
}

func (r *TaskRegistry) GetTraceContext(traceID string) *TraceContext {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.traceContexts[traceID]
}

func (r *TaskRegistry) GetTraceContextByTaskID(taskID string) *TraceContext {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, tc := range r.traceContexts {
		if tc.RootTaskID == taskID {
			return tc
		}
	}
	return nil
}

func (r *TaskRegistry) UnregisterTraceContext(traceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.traceContexts, traceID)
}

func (r *TaskRegistry) RegisterTaskContext(taskID string, ctx *TaskContext) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.taskContexts[taskID] = ctx
}

func (r *TaskRegistry) GetTaskContext(taskID string) *TaskContext {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.taskContexts[taskID]
}

func (r *TaskRegistry) UnregisterTaskContext(taskID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.taskContexts, taskID)
}

func (r *TaskRegistry) RegisterTodoList(taskID string, tl *TodoList) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.todoLists[taskID] = tl
}

func (r *TaskRegistry) GetTodoList(taskID string) *TodoList {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.todoLists[taskID]
}

func (r *TaskRegistry) UnregisterTodoList(taskID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.todoLists, taskID)
}

func (r *TaskRegistry) GetAllTraceContexts() map[string]*TraceContext {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]*TraceContext)
	for k, v := range r.traceContexts {
		result[k] = v
	}
	return result
}

func (r *TaskRegistry) GetAllTodoLists() map[string]*TodoList {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]*TodoList)
	for k, v := range r.todoLists {
		result[k] = v
	}
	return result
}
