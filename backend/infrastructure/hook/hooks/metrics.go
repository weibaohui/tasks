/**
 * 指标 Hook 实现
 */
package hooks

import (
	"sync"
	"time"

	"github.com/weibh/taskmanager/domain"
	"go.uber.org/zap"
)

// MetricsCollector 指标收集器
type MetricsCollector struct {
	mu           sync.RWMutex
	counters     map[string]int64
	gauges       map[string]interface{}
	histograms   map[string][]time.Duration
	lastDuration map[string]time.Time
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		counters:     make(map[string]int64),
		gauges:       make(map[string]interface{}),
		histograms:   make(map[string][]time.Duration),
		lastDuration: make(map[string]time.Time),
	}
}

// Increment 增加计数器
func (m *MetricsCollector) Increment(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name]++
}

// Set 设置 gauge 值
func (m *MetricsCollector) Set(name string, val interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = val
}

// Record 记录直方图值
func (m *MetricsCollector) Record(name string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.histograms[name] = append(m.histograms[name], duration)
}

// SetStart 设置开始时间
func (m *MetricsCollector) SetStart(name string, t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastDuration[name] = t
}

// GetCounter 获取计数器值
func (m *MetricsCollector) GetCounter(name string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.counters[name]
}

// GetGauge 获取 gauge 值
func (m *MetricsCollector) GetGauge(name string) interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.gauges[name]
}

// MetricsHook 收集指标
type MetricsHook struct {
	*domain.BaseHook
	collector  *MetricsCollector
	startTimes map[string]time.Time
	mu         sync.Mutex
	logger     *zap.Logger
}

// NewMetricsHook 创建指标 Hook
func NewMetricsHook(logger *zap.Logger) *MetricsHook {
	return &MetricsHook{
		BaseHook:   domain.NewBaseHook("metrics", 50, domain.HookTypeLLM),
		collector:  NewMetricsCollector(),
		startTimes: make(map[string]time.Time),
		logger:     logger,
	}
}

// PreLLMCall 记录 LLM 调用开始
func (h *MetricsHook) PreLLMCall(ctx *domain.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error) {
	h.mu.Lock()
	h.startTimes["llm"] = time.Now()
	h.mu.Unlock()

	h.collector.Increment("llm_call_total")
	h.collector.Set("llm_model", callCtx.Model)

	return callCtx, nil
}

// PostLLMCall 记录 LLM 调用完成
func (h *MetricsHook) PostLLMCall(ctx *domain.HookContext, callCtx *domain.LLMCallContext, resp *domain.LLMResponse) (*domain.LLMResponse, error) {
	h.mu.Lock()
	if start, ok := h.startTimes["llm"]; ok {
		h.collector.Record("llm_call_duration", time.Since(start))
		delete(h.startTimes, "llm")
	}
	h.mu.Unlock()

	h.collector.Set("llm_prompt_tokens", resp.Usage.PromptTokens)
	h.collector.Set("llm_completion_tokens", resp.Usage.CompletionTokens)
	h.collector.Set("llm_total_tokens", resp.Usage.TotalTokens)

	h.logger.Info("LLM call completed",
		zap.Int("prompt_tokens", resp.Usage.PromptTokens),
		zap.Int("completion_tokens", resp.Usage.CompletionTokens),
		zap.Int("total_tokens", resp.Usage.TotalTokens))

	return resp, nil
}

// PreToolCall 记录工具调用开始
func (h *MetricsHook) PreToolCall(ctx *domain.HookContext, callCtx *domain.ToolCallContext) (*domain.ToolCallContext, error) {
	h.mu.Lock()
	h.startTimes["tool_"+callCtx.ToolName] = time.Now()
	h.mu.Unlock()

	h.collector.Increment("tool_call_total")
	h.collector.Set("tool_name", callCtx.ToolName)

	return callCtx, nil
}

// PostToolCall 记录工具调用完成
func (h *MetricsHook) PostToolCall(ctx *domain.HookContext, callCtx *domain.ToolCallContext, result *domain.ToolExecutionResult) (*domain.ToolExecutionResult, error) {
	h.mu.Lock()
	if start, ok := h.startTimes["tool_"+callCtx.ToolName]; ok {
		h.collector.Record("tool_call_duration", time.Since(start))
		delete(h.startTimes, "tool_"+callCtx.ToolName)
	}
	h.mu.Unlock()

	if result.Success {
		h.collector.Increment("tool_call_success")
	} else {
		h.collector.Increment("tool_call_failure")
	}

	return result, nil
}

// OnToolError 记录工具错误
func (h *MetricsHook) OnToolError(ctx *domain.HookContext, callCtx *domain.ToolCallContext, err error) (*domain.ToolExecutionResult, error) {
	h.collector.Increment("tool_call_error")
	return &domain.ToolExecutionResult{Success: false, Error: err}, nil
}

// GetCollector 获取指标收集器
func (h *MetricsHook) GetCollector() *MetricsCollector {
	return h.collector
}
