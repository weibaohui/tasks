/**
 * Hook 注册表实现
 */
package hook

import (
	"errors"
	"fmt"
	"sync"

	"github.com/weibh/taskmanager/domain"
)

// Registry Hook 注册表
type Registry interface {
	Register(hook domain.Hook) error
	Unregister(name string) error
	Get(name string) domain.Hook
	List() []domain.Hook
	ListByType(hookType domain.HookType) []domain.Hook
	Enable(name string) error
	Disable(name string) error
	Clear()
}

// SimpleRegistry 简单注册表实现
type SimpleRegistry struct {
	mu    sync.RWMutex
	hooks map[string]domain.Hook
}

// NewRegistry 创建注册表
func NewRegistry() Registry {
	return &SimpleRegistry{
		hooks: make(map[string]domain.Hook),
	}
}

func (r *SimpleRegistry) Register(hook domain.Hook) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if hook == nil {
		return errors.New("hook cannot be nil")
	}

	if _, exists := r.hooks[hook.Name()]; exists {
		return fmt.Errorf("hook %s already registered", hook.Name())
	}

	r.hooks[hook.Name()] = hook
	return nil
}

func (r *SimpleRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.hooks[name]; !exists {
		return fmt.Errorf("hook %s not found", name)
	}

	delete(r.hooks, name)
	return nil
}

func (r *SimpleRegistry) Get(name string) domain.Hook {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.hooks[name]
}

func (r *SimpleRegistry) List() []domain.Hook {
	r.mu.RLock()
	defer r.mu.RUnlock()

	hooks := make([]domain.Hook, 0, len(r.hooks))
	for _, hook := range r.hooks {
		hooks = append(hooks, hook)
	}
	return hooks
}

func (r *SimpleRegistry) ListByType(hookType domain.HookType) []domain.Hook {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var hooks []domain.Hook
	for _, hook := range r.hooks {
		if hook.HookType() == hookType {
			hooks = append(hooks, hook)
		}
	}
	return hooks
}

func (r *SimpleRegistry) Enable(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	hook, exists := r.hooks[name]
	if !exists {
		return fmt.Errorf("hook %s not found", name)
	}
	hook.SetEnabled(true)
	return nil
}

func (r *SimpleRegistry) Disable(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	hook, exists := r.hooks[name]
	if !exists {
		return fmt.Errorf("hook %s not found", name)
	}
	hook.SetEnabled(false)
	return nil
}

func (r *SimpleRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks = make(map[string]domain.Hook)
}
