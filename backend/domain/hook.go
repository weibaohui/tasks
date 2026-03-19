/**
 * Hook 系统接口定义
 */
package domain

import "context"

// TaskHook 任务钩子接口
type TaskHook interface {
	// Name 返回钩子名称
	Name() string
	// OnTaskCreated 任务创建时调用
	OnTaskCreated(ctx context.Context, task *Task) error
	// OnTaskStarted 任务开始时调用
	OnTaskStarted(ctx context.Context, task *Task) error
	// OnTaskCompleted 任务完成时调用
	OnTaskCompleted(ctx context.Context, task *Task) error
	// OnTaskFailed 任务失败时调用
	OnTaskFailed(ctx context.Context, task *Task, err error) error
	// OnTaskCancelled 任务取消时调用
	OnTaskCancelled(ctx context.Context, task *Task) error
	// OnTaskProgressUpdated 任务进度更新时调用
	OnTaskProgressUpdated(ctx context.Context, task *Task) error
}

// TaskHooks 任务钩子集合
type TaskHooks []TaskHook

// HookRegistry 钩子注册表接口
type HookRegistry interface {
	// Register 注册钩子
	Register(hook TaskHook) error
	// Unregister 取消注册
	Unregister(name string) error
	// GetHooks 获取所有钩子
	GetHooks() TaskHooks
}

// HookExecutor 钩子执行器
type HookExecutor struct {
	registry HookRegistry
}

func NewHookExecutor(registry HookRegistry) *HookExecutor {
	return &HookExecutor{registry: registry}
}

// ExecuteOnTaskCreated 执行 OnTaskCreated 钩子
func (e *HookExecutor) ExecuteOnTaskCreated(ctx context.Context, task *Task) error {
	for _, hook := range e.registry.GetHooks() {
		if err := hook.OnTaskCreated(ctx, task); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteOnTaskStarted 执行 OnTaskStarted 钩子
func (e *HookExecutor) ExecuteOnTaskStarted(ctx context.Context, task *Task) error {
	for _, hook := range e.registry.GetHooks() {
		if err := hook.OnTaskStarted(ctx, task); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteOnTaskCompleted 执行 OnTaskCompleted 钩子
func (e *HookExecutor) ExecuteOnTaskCompleted(ctx context.Context, task *Task) error {
	for _, hook := range e.registry.GetHooks() {
		if err := hook.OnTaskCompleted(ctx, task); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteOnTaskFailed 执行 OnTaskFailed 钩子
func (e *HookExecutor) ExecuteOnTaskFailed(ctx context.Context, task *Task, err error) error {
	for _, hook := range e.registry.GetHooks() {
		if err := hook.OnTaskFailed(ctx, task, err); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteOnTaskCancelled 执行 OnTaskCancelled 钩子
func (e *HookExecutor) ExecuteOnTaskCancelled(ctx context.Context, task *Task) error {
	for _, hook := range e.registry.GetHooks() {
		if err := hook.OnTaskCancelled(ctx, task); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteOnTaskProgressUpdated 执行 OnTaskProgressUpdated 钩子
func (e *HookExecutor) ExecuteOnTaskProgressUpdated(ctx context.Context, task *Task) error {
	for _, hook := range e.registry.GetHooks() {
		if err := hook.OnTaskProgressUpdated(ctx, task); err != nil {
			return err
		}
	}
	return nil
}
