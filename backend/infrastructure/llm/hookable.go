/**
 * Hookable LLM Provider - 为 LLM Provider 添加 Hook 支持
 */
package llm

import (
	"context"
	"fmt"

	"github.com/weibh/taskmanager/domain"
)

// HookManagerInterface Hook 管理器接口
type HookManagerInterface interface {
	PreLLMCall(ctx context.Context, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error)
	PostLLMCall(ctx context.Context, callCtx *domain.LLMCallContext, resp *domain.LLMResponse) (*domain.LLMResponse, error)
}

// HookableProvider 包装普通 Provider，添加 Hook 支持
type HookableProvider struct {
	wrapped  LLMProvider
	hookMgr HookManagerInterface
}

// NewHookableProvider 创建 Hookable Provider
func NewHookableProvider(wrapped LLMProvider) *HookableProvider {
	return &HookableProvider{wrapped: wrapped}
}

// SetHookManager 设置 Hook 管理器
func (p *HookableProvider) SetHookManager(manager HookManagerInterface) {
	p.hookMgr = manager
}

// Generate 生成文本（带 Hook 支持）
func (p *HookableProvider) Generate(ctx context.Context, prompt string) (string, error) {
	// 1. Pre Hook
	if p.hookMgr != nil {
		callCtx := &domain.LLMCallContext{
			Prompt: prompt,
			Model:  p.wrapped.Name(),
		}
		modifiedCtx, err := p.hookMgr.PreLLMCall(ctx, callCtx)
		if err != nil {
			return "", fmt.Errorf("PreLLMCall hook failed: %w", err)
		}
		prompt = modifiedCtx.Prompt
	}

	// 2. Actual LLM Call
	response, err := p.wrapped.Generate(ctx, prompt)
	if err != nil {
		return "", err
	}

	// 3. Post Hook
	if p.hookMgr != nil {
		resp := &domain.LLMResponse{
			Content: response,
			Model:   p.wrapped.Name(),
			Usage:   p.getUsage(),
		}
		modifiedResp, err := p.hookMgr.PostLLMCall(ctx, &domain.LLMCallContext{Prompt: prompt}, resp)
		if err != nil {
			return "", fmt.Errorf("PostLLMCall hook failed: %w", err)
		}
		response = modifiedResp.Content
	}

	return response, nil
}

// getUsage 获取底层 Provider 的 Usage
func (p *HookableProvider) getUsage() domain.Usage {
	if openAIProvider, ok := p.wrapped.(*OpenAIProvider); ok {
		usage := openAIProvider.GetLastUsage()
		return domain.Usage{
			PromptTokens:     usage.PromptTokens,
			CompletionTokens: usage.CompletionTokens,
			TotalTokens:     usage.TotalTokens,
		}
	}
	// 其他 Provider 暂时返回空 Usage
	return domain.Usage{}
}

// GenerateSubTasks 生成子任务计划（带 Hook 支持）
func (p *HookableProvider) GenerateSubTasks(ctx context.Context, taskName string, taskDesc string, depth int, maxDepth int) (*SubTaskPlan, error) {
	// 1. Pre Hook - 生成一个特殊的 prompt
	prompt := SubTaskPrompt(taskName, taskDesc, depth, maxDepth)

	if p.hookMgr != nil {
		callCtx := &domain.LLMCallContext{
			Prompt:      prompt,
			Model:       p.wrapped.Name(),
			SessionID:   "",
			TraceID:     "",
		}
		modifiedCtx, err := p.hookMgr.PreLLMCall(ctx, callCtx)
		if err != nil {
			return nil, fmt.Errorf("PreLLMCall hook failed: %w", err)
		}
		prompt = modifiedCtx.Prompt
	}

	// 2. Actual LLM Call
	response, err := p.wrapped.Generate(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// 3. Post Hook
	if p.hookMgr != nil {
		resp := &domain.LLMResponse{
			Content: response,
			Model:   p.wrapped.Name(),
			Usage:   p.getUsage(),
		}
		modifiedResp, err := p.hookMgr.PostLLMCall(ctx, &domain.LLMCallContext{Prompt: prompt}, resp)
		if err != nil {
			return nil, fmt.Errorf("PostLLMCall hook failed: %w", err)
		}
		response = modifiedResp.Content
	}

	// 4. 解析 YAML
	yamlContent := ExtractYAML(response)
	plan, err := TryParseYAML(yamlContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return plan, nil
}

// Name 返回 provider 名称
func (p *HookableProvider) Name() string {
	return p.wrapped.Name()
}
