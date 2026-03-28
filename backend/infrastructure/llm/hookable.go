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
	PreLLMCall(ctx *domain.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error)
	PostLLMCall(ctx *domain.HookContext, callCtx *domain.LLMCallContext, resp *domain.LLMResponse) (*domain.LLMResponse, error)
}

// HookableProvider 包装普通 Provider，添加 Hook 支持
type HookableProvider struct {
	wrapped LLMProvider
	hookMgr HookManagerInterface
	// 用于 GenerateSubTasks 的上下文信息
	sessionID   string
	agentCode   string
	userCode    string
	channelCode string
	traceID     string
}

// NewHookableProvider 创建 Hookable Provider
func NewHookableProvider(wrapped LLMProvider) *HookableProvider {
	return &HookableProvider{wrapped: wrapped}
}

// SetHookManager 设置 Hook 管理器
func (p *HookableProvider) SetHookManager(manager HookManagerInterface) {
	p.hookMgr = manager
}

// SetContextInfo 设置上下文信息（session_key, agent_code 等）
// 这些信息会在 PreLLMCall 时传递给 hook
func (p *HookableProvider) SetContextInfo(sessionID, agentCode, userCode, channelCode, traceID string) {
	p.sessionID = sessionID
	p.agentCode = agentCode
	p.userCode = userCode
	p.channelCode = channelCode
	p.traceID = traceID
}

// Generate 生成文本（带 Hook 支持）
func (p *HookableProvider) Generate(ctx context.Context, prompt string) (string, error) {
	// 1. Pre Hook
	if p.hookMgr != nil {
		hookCtx := domain.NewHookContext(ctx)
		callCtx := &domain.LLMCallContext{
			Prompt: prompt,
			Model:  p.wrapped.Name(),
		}
		modifiedCtx, err := p.hookMgr.PreLLMCall(hookCtx, callCtx)
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
		hookCtx := domain.NewHookContext(ctx)
		resp := &domain.LLMResponse{
			Content: response,
			Model:   p.wrapped.Name(),
			Usage:   p.getUsage(),
		}
		modifiedResp, err := p.hookMgr.PostLLMCall(hookCtx, &domain.LLMCallContext{Prompt: prompt}, resp)
		if err != nil {
			return "", fmt.Errorf("PostLLMCall hook failed: %w", err)
		}
		response = modifiedResp.Content
	}

	return response, nil
}

// getUsage 获取底层 Provider 的 Usage
func (p *HookableProvider) getUsage() domain.Usage {
	usage := p.wrapped.GetLastUsage()
	return domain.Usage{
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
	}
}

// GenerateSubTasks 生成子任务计划（带 Hook 支持）
func (p *HookableProvider) GenerateSubTasks(ctx context.Context, taskName string, taskDesc string, depth int, maxDepth int) (*SubTaskPlan, error) {
	// 1. Pre Hook - 生成一个特殊的 prompt
	prompt := SubTaskPrompt(taskName, taskDesc, depth, maxDepth)

	// 构建 call metadata
	callMetadata := make(map[string]string)
	if p.sessionID != "" {
		callMetadata["session_key"] = p.sessionID
	}
	if p.agentCode != "" {
		callMetadata["agent_code"] = p.agentCode
	}
	if p.userCode != "" {
		callMetadata["user_code"] = p.userCode
	}
	if p.channelCode != "" {
		callMetadata["channel_code"] = p.channelCode
	}

	if p.hookMgr != nil {
		hookCtx := domain.NewHookContext(ctx)
		callCtx := &domain.LLMCallContext{
			Prompt:    prompt,
			Model:     p.wrapped.Name(),
			SessionID: p.sessionID,
			TraceID:   p.traceID,
			Metadata:  callMetadata,
		}
		modifiedCtx, err := p.hookMgr.PreLLMCall(hookCtx, callCtx)
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
		hookCtx := domain.NewHookContext(ctx)
		resp := &domain.LLMResponse{
			Content: response,
			Model:   p.wrapped.Name(),
			Usage:   p.getUsage(),
		}
		callCtx := &domain.LLMCallContext{
			Prompt:    prompt,
			SessionID: p.sessionID,
			TraceID:   p.traceID,
			Metadata:  callMetadata,
		}
		modifiedResp, err := p.hookMgr.PostLLMCall(hookCtx, callCtx, resp)
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

// GetLastUsage 返回上次调用的 token 使用量
func (p *HookableProvider) GetLastUsage() Usage {
	return p.wrapped.GetLastUsage()
}

// GenerateWithTools 生成文本，支持工具调用（直接委托给 wrapped provider，暂不添加 hook 支持）
func (p *HookableProvider) GenerateWithTools(ctx context.Context, prompt string, tools []*ToolRegistry, maxIterations int) (string, []ToolCall, error) {
	return p.wrapped.GenerateWithTools(ctx, prompt, tools, maxIterations)
}
