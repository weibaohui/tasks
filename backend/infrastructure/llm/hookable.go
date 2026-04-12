/**
 * Hookable LLM Provider - 为 LLM Provider 添加 Hook 支持
 */
package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/weibh/taskmanager/domain"
)

// HookManagerInterface Hook 管理器接口
type HookManagerInterface interface {
	PreLLMCall(ctx *domain.HookContext, callCtx *domain.LLMCallContext) (*domain.LLMCallContext, error)
	PostLLMCall(ctx *domain.HookContext, callCtx *domain.LLMCallContext, resp *domain.LLMResponse) (*domain.LLMResponse, error)
}

// HookableProvider 包装普通 Provider，添加 Hook 支持
type HookableProvider struct {
	wrapped domain.LLMClient
	hookMgr HookManagerInterface
	// 用于 GenerateSubTasks 的上下文信息
	sessionID   string
	agentCode   string
	userCode    string
	channelCode string
	traceID     string
	// 保存 Pre 阶段的上下文，供 Post 阶段复用
	preHookCtx *domain.HookContext
	preCallCtx *domain.LLMCallContext
}

// NewHookableProvider 创建 Hookable Provider
func NewHookableProvider(wrapped domain.LLMClient) *HookableProvider {
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
	// 1. Pre Hook - 保存上下文供 Post 复用
	if p.hookMgr != nil {
		p.preHookCtx = domain.NewHookContext(ctx)
		p.preCallCtx = &domain.LLMCallContext{
			Prompt:    prompt,
			Model:     p.wrapped.Name(),
			SessionID: p.sessionID,
			TraceID:   p.traceID,
		}
		modifiedCtx, err := p.hookMgr.PreLLMCall(p.preHookCtx, p.preCallCtx)
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

	// 3. Post Hook - 复用 Pre 阶段的上下文
	if p.hookMgr != nil {
		resp := &domain.LLMResponse{
			Content: response,
			Model:   p.wrapped.Name(),
			Usage:   p.getUsage(),
		}
		// 更新 preCallCtx 中的 prompt 和 response
		p.preCallCtx.Prompt = prompt
		modifiedResp, err := p.hookMgr.PostLLMCall(p.preHookCtx, p.preCallCtx, resp)
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
func (p *HookableProvider) GenerateSubTasks(ctx context.Context, taskName string, taskDesc string, depth int, maxDepth int) (*domain.SubTaskPlan, error) {
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
	channelTypeFromSession, chatIDFromSession := parseSessionKey(p.sessionID)
	if channelTypeFromSession != "" {
		callMetadata["channel_type"] = channelTypeFromSession
	}
	if chatIDFromSession != "" {
		callMetadata["chat_id"] = chatIDFromSession
	}

	// Pre Hook - 保存上下文供 Post 复用
	if p.hookMgr != nil {
		p.preHookCtx = domain.NewHookContext(ctx)
		p.preCallCtx = &domain.LLMCallContext{
			Prompt:    prompt,
			Model:     p.wrapped.Name(),
			SessionID: p.sessionID,
			TraceID:   p.traceID,
			Metadata:  callMetadata,
		}
		modifiedCtx, err := p.hookMgr.PreLLMCall(p.preHookCtx, p.preCallCtx)
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

	// 3. Post Hook - 复用 Pre 阶段的上下文
	if p.hookMgr != nil {
		resp := &domain.LLMResponse{
			Content: response,
			Model:   p.wrapped.Name(),
			Usage:   p.getUsage(),
		}
		// 更新 preCallCtx 中的 prompt
		p.preCallCtx.Prompt = prompt
		modifiedResp, err := p.hookMgr.PostLLMCall(p.preHookCtx, p.preCallCtx, resp)
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
func (p *HookableProvider) GetLastUsage() domain.Usage {
	return p.wrapped.GetLastUsage()
}

// GenerateWithTools 生成文本，支持工具调用（直接委托给 wrapped provider，暂不添加 hook 支持）
func (p *HookableProvider) GenerateWithTools(ctx context.Context, prompt string, tools []*domain.ToolRegistry, maxIterations int) (string, []domain.ToolCall, error) {
	return p.wrapped.GenerateWithTools(ctx, prompt, tools, maxIterations)
}

// parseSessionKey 解析 session_key，提取渠道类型与会话 ID
func parseSessionKey(sessionKey string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(sessionKey), ":", 2)
	if len(parts) != 2 {
		return "", ""
	}
	channelType := strings.TrimSpace(parts[0])
	chatID := strings.TrimSpace(parts[1])
	if channelType == "" || chatID == "" {
		return "", ""
	}
	return channelType, chatID
}
