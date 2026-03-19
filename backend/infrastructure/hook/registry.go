/**
 * HookRegistry 钩子注册表
 * 管理任务钩子的注册和获取
 */
package hook

import (
	"github.com/weibh/taskmanager/domain"
	"sync"
)

// DefaultHookRegistry 默认钩子注册表
type DefaultHookRegistry struct {
	mu    sync.RWMutex
	hooks domain.TaskHooks
}

// NewDefaultHookRegistry 创建钩子注册表
func NewDefaultHookRegistry() *DefaultHookRegistry {
	return &DefaultHookRegistry{
		hooks: make(domain.TaskHooks, 0),
	}
}

// Register 注册钩子
func (r *DefaultHookRegistry) Register(hook domain.TaskHook) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks = append(r.hooks, hook)
	return nil
}

// Unregister 取消注册
func (r *DefaultHookRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, hook := range r.hooks {
		if hook.Name() == name {
			r.hooks = append(r.hooks[:i], r.hooks[i+1:]...)
			return nil
		}
	}
	return nil
}

// GetHooks 获取所有钩子
func (r *DefaultHookRegistry) GetHooks() domain.TaskHooks {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.hooks
}
