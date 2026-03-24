/**
 * Hook Executor 单元测试
 */
package hook

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/weibh/taskmanager/domain"
	"go.uber.org/zap"
)

func TestExecutor_ExecutePreLLMCall(t *testing.T) {
	registry := NewRegistry()
	logger, _ := zap.NewDevelopment()
	executor := NewExecutor(registry, logger)

	// 注册一个修改 prompt 的 Hook
	registry.Register(&mockLLMHook{
		name:     "modifier",
		priority: 10,
		enabled:  true,
		preCallFn: func(ctx *domain.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
			callCtx.Prompt = callCtx.Prompt + " [modified]"
			return callCtx, nil
		},
	})

	// Test: 执行 PreLLMCall
	ctx := domain.NewHookContext(context.Background())
	callCtx := &domain.LLMCallContext{Prompt: "original"}
	result, err := executor.ExecutePreLLMCall(ctx, callCtx)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Prompt != "original [modified]" {
		t.Fatalf("expected prompt 'original [modified]', got '%s'", result.Prompt)
	}
}

func TestExecutor_PriorityOrdering(t *testing.T) {
	registry := NewRegistry()
	logger, _ := zap.NewDevelopment()
	executor := NewExecutor(registry, logger)

	var callOrder []string
	var mu sync.Mutex

	// 注册多个 hooks，优先级不同
	registry.Register(&mockLLMHook{
		name:     "hook-priority-20",
		priority: 20,
		enabled:  true,
		preCallFn: func(ctx *domain.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
			mu.Lock()
			callOrder = append(callOrder, "hook-priority-20")
			mu.Unlock()
			return callCtx, nil
		},
	})
	registry.Register(&mockLLMHook{
		name:     "hook-priority-10",
		priority: 10,
		enabled:  true,
		preCallFn: func(ctx *domain.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
			mu.Lock()
			callOrder = append(callOrder, "hook-priority-10")
			mu.Unlock()
			return callCtx, nil
		},
	})
	registry.Register(&mockLLMHook{
		name:     "hook-priority-30",
		priority: 30,
		enabled:  true,
		preCallFn: func(ctx *domain.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
			mu.Lock()
			callOrder = append(callOrder, "hook-priority-30")
			mu.Unlock()
			return callCtx, nil
		},
	})

	ctx := domain.NewHookContext(context.Background())
	executor.ExecutePreLLMCall(ctx, &domain.LLMCallContext{Prompt: "test"})

	// Test: 按优先级顺序执行
	expected := []string{"hook-priority-10", "hook-priority-20", "hook-priority-30"}
	mu.Lock()
	defer mu.Unlock()
	if len(callOrder) != len(expected) {
		t.Fatalf("expected %d calls, got %d", len(expected), len(callOrder))
	}
	for i, e := range expected {
		if callOrder[i] != e {
			t.Fatalf("at index %d: expected '%s', got '%s'", i, e, callOrder[i])
		}
	}
}

func TestExecutor_DisabledHookSkipped(t *testing.T) {
	registry := NewRegistry()
	logger, _ := zap.NewDevelopment()
	executor := NewExecutor(registry, logger)

	var callOrder []string
	var mu sync.Mutex

	registry.Register(&mockLLMHook{
		name:     "enabled-hook",
		priority: 10,
		enabled:  true,
		preCallFn: func(ctx *domain.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
			mu.Lock()
			callOrder = append(callOrder, "enabled-hook")
			mu.Unlock()
			return callCtx, nil
		},
	})
	registry.Register(&mockLLMHook{
		name:     "disabled-hook",
		priority: 5, // 更高优先级但被禁用
		enabled:  false,
		preCallFn: func(ctx *domain.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
			mu.Lock()
			callOrder = append(callOrder, "disabled-hook")
			mu.Unlock()
			return callCtx, nil
		},
	})

	ctx := domain.NewHookContext(context.Background())
	executor.ExecutePreLLMCall(ctx, &domain.LLMCallContext{Prompt: "test"})

	// Test: 禁用的 hook 应被跳过
	mu.Lock()
	defer mu.Unlock()
	if len(callOrder) != 1 {
		t.Fatalf("expected 1 call, got %d", len(callOrder))
	}
	if callOrder[0] != "enabled-hook" {
		t.Fatalf("expected 'enabled-hook', got '%s'", callOrder[0])
	}
}

func TestExecutor_ErrorHandling_Continue(t *testing.T) {
	registry := NewRegistry()
	logger, _ := zap.NewDevelopment()
	executor := NewExecutor(registry, logger)
	executor.SetErrorStrategy(ErrorStrategyContinue)

	registry.Register(&mockLLMHook{
		name:     "error-hook",
		priority: 10,
		enabled:  true,
		preCallFn: func(ctx *domain.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
			return nil, errors.New("hook error")
		},
	})
	registry.Register(&mockLLMHook{
		name:     "after-error-hook",
		priority: 20,
		enabled:  true,
		preCallFn: func(ctx *domain.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
			return callCtx, nil
		},
	})

	ctx := domain.NewHookContext(context.Background())
	result, err := executor.ExecutePreLLMCall(ctx, &domain.LLMCallContext{Prompt: "test"})

	// Test: Continue 模式下，错误后继续执行
	if err != nil {
		t.Fatalf("expected no returned error in continue mode, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result to be nil (error hook returned nil)")
	}
	if !ctx.HasErrors() {
		t.Fatal("expected errors to be recorded in context")
	}
}

func TestExecutor_ErrorHandling_StopOnFirst(t *testing.T) {
	registry := NewRegistry()
	logger, _ := zap.NewDevelopment()
	executor := NewExecutor(registry, logger)
	executor.SetErrorStrategy(ErrorStrategyStopOnFirst)

	registry.Register(&mockLLMHook{
		name:     "error-hook",
		priority: 10,
		enabled:  true,
		preCallFn: func(ctx *domain.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
			return nil, errors.New("hook error")
		},
	})
	registry.Register(&mockLLMHook{
		name:     "after-error-hook",
		priority: 20,
		enabled:  true,
		preCallFn: func(ctx *domain.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
			return callCtx, nil
		},
	})

	ctx := domain.NewHookContext(context.Background())
	_, err := executor.ExecutePreLLMCall(ctx, &domain.LLMCallContext{Prompt: "test"})

	// Test: StopOnFirst 模式下，遇到错误立即返回
	if err == nil {
		t.Fatal("expected error in stop on first mode")
	}
}

func TestExecutor_HookContextPropagation(t *testing.T) {
	registry := NewRegistry()
	logger, _ := zap.NewDevelopment()
	executor := NewExecutor(registry, logger)

	registry.Register(&mockLLMHook{
		name:     "hook-1",
		priority: 10,
		enabled:  true,
		preCallFn: func(ctx *domain.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
			ctx.SetMetadata("key1", "value1")
			return callCtx, nil
		},
	})
	registry.Register(&mockLLMHook{
		name:     "hook-2",
		priority: 20,
		enabled:  true,
		preCallFn: func(ctx *domain.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
			// 应该能访问 hook-1 设置的值
			val := ctx.GetMetadata("key1")
			if val != "value1" {
				t.Errorf("expected 'value1', got '%v'", val)
			}
			return callCtx, nil
		},
	})

	ctx := domain.NewHookContext(context.Background())
	executor.ExecutePreLLMCall(ctx, &domain.LLMCallContext{Prompt: "test"})

	// Test: Hook 列表应该包含所有执行的 hook
	hooks := ctx.GetHooks()
	if len(hooks) != 2 {
		t.Fatalf("expected 2 hooks, got %d", len(hooks))
	}
}

func TestExecutor_PostLLMCall(t *testing.T) {
	registry := NewRegistry()
	logger, _ := zap.NewDevelopment()
	executor := NewExecutor(registry, logger)

	// 注册一个修改响应的 Hook
	registry.Register(&mockLLMHook{
		name:     "response-modifier",
		priority: 10,
		enabled:  true,
		postCallFn: func(ctx *domain.HookContext, callCtx *domain.LLMCallContext, resp *domain.LLMResponse) (*domain.LLMResponse, error) {
			resp.Content = resp.Content + " [modified]"
			return resp, nil
		},
	})

	ctx := domain.NewHookContext(context.Background())
	callCtx := &domain.LLMCallContext{Prompt: "test"}
	resp := &domain.LLMResponse{Content: "original response"}

	result, err := executor.ExecutePostLLMCall(ctx, callCtx, resp)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Content != "original response [modified]" {
		t.Fatalf("expected 'original response [modified]', got '%s'", result.Content)
	}
}
