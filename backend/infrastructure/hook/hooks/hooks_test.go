/**
 * 内置 Hooks 单元测试
 */
package hooks

import (
	"context"
	"testing"
	"time"

	"github.com/weibh/taskmanager/domain"
	"go.uber.org/zap"
)

func TestLoggingHook(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	h := NewLoggingHook(logger)

	if h.Name() != "logging" {
		t.Errorf("expected name 'logging', got '%s'", h.Name())
	}
	if h.Priority() != 100 {
		t.Errorf("expected priority 100, got %d", h.Priority())
	}
	if !h.Enabled() {
		t.Error("expected enabled")
	}
}

func TestLoggingHook_PreLLMCall(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	h := NewLoggingHook(logger)

	ctx := domain.NewHookContext(context.Background())
	callCtx := &domain.LLMCallContext{
		Prompt:   "test prompt",
		Model:    "gpt-4",
		SessionID: "sess-123",
		TraceID:  "trace-456",
	}

	result, err := h.PreLLMCall(ctx, callCtx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != callCtx {
		t.Error("expected same context returned")
	}
}

func TestLoggingHook_PostLLMCall(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	h := NewLoggingHook(logger)

	ctx := domain.NewHookContext(context.Background())
	callCtx := &domain.LLMCallContext{Prompt: "test"}
	resp := &domain.LLMResponse{
		Content: "test response",
		Usage: domain.Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
		FinishReason: "stop",
	}

	result, err := h.PostLLMCall(ctx, callCtx, resp)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != resp {
		t.Error("expected same response returned")
	}
}

func TestMetricsHook(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	h := NewMetricsHook(logger)

	if h.Name() != "metrics" {
		t.Errorf("expected name 'metrics', got '%s'", h.Name())
	}
	if h.Priority() != 50 {
		t.Errorf("expected priority 50, got %d", h.Priority())
	}
	if !h.Enabled() {
		t.Error("expected enabled")
	}
}

func TestMetricsHook_PreLLMCall(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	h := NewMetricsHook(logger)

	ctx := domain.NewHookContext(context.Background())
	callCtx := &domain.LLMCallContext{
		Prompt: "test prompt",
		Model:  "gpt-4",
	}

	result, err := h.PreLLMCall(ctx, callCtx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != callCtx {
		t.Error("expected same context returned")
	}

	collector := h.GetCollector()
	if collector.GetCounter("llm_call_total") != 1 {
		t.Errorf("expected 1 llm_call_total, got %d", collector.GetCounter("llm_call_total"))
	}
}

func TestMetricsHook_PostLLMCall(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	h := NewMetricsHook(logger)

	ctx := domain.NewHookContext(context.Background())
	callCtx := &domain.LLMCallContext{Prompt: "test"}
	resp := &domain.LLMResponse{
		Content: "test response",
		Usage: domain.Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	time.Sleep(10 * time.Millisecond)

	result, err := h.PostLLMCall(ctx, callCtx, resp)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != resp {
		t.Error("expected same response returned")
	}

	collector := h.GetCollector()
	if collector.GetGauge("llm_total_tokens") != 30 {
		t.Errorf("expected 30 llm_total_tokens, got %v", collector.GetGauge("llm_total_tokens"))
	}
}

func TestRateLimitHook(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	h := NewRateLimitHook(10, 20, logger) // 10 req/s, burst 20

	if !h.Enabled() {
		t.Error("expected enabled")
	}
	if h.Priority() != 5 {
		t.Errorf("expected priority 5, got %d", h.Priority())
	}
}

func TestRateLimitHook_Allow(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	h := NewRateLimitHook(10, 20, logger) // 10 req/s, burst 20

	ctx := domain.NewHookContext(context.Background())
	callCtx := &domain.LLMCallContext{Prompt: "test"}

	// Test: 前 20 个请求应该通过
	for i := 0; i < 20; i++ {
		_, err := h.PreLLMCall(ctx, callCtx)
		if err != nil {
			t.Fatalf("request %d: expected no error, got %v", i, err)
		}
	}

	// Test: 第 21 个请求应该被限流
	_, err := h.PreLLMCall(ctx, callCtx)
	if err == nil {
		t.Fatal("expected rate limit error")
	}
	if err != ErrRateLimited {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
}

func TestRateLimitHook_SetLimit(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	h := NewRateLimitHook(100, 100, logger) // 高 limit，burst 100

	ctx := domain.NewHookContext(context.Background())
	callCtx := &domain.LLMCallContext{Prompt: "test"}

	// 快速消耗一些令牌
	for i := 0; i < 5; i++ {
		_, err := h.PreLLMCall(ctx, callCtx)
		if err != nil {
			t.Fatalf("request %d: expected no error, got %v", i, err)
		}
	}

	// 降低 limit 到 1
	h.SetLimit(1)

	// 等待令牌恢复（1 req/s，所以 1 秒后会有 1 个令牌）
	time.Sleep(1100 * time.Millisecond)

	// 现在应该可以通过（因为令牌已恢复）
	_, err := h.PreLLMCall(ctx, callCtx)
	if err != nil {
		t.Fatalf("expected no error after wait, got %v", err)
	}
}
