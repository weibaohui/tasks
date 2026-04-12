package domain

import "context"

// LLMClient LLM 生成客户端接口
type LLMClient interface {
	// Generate 生成文本
	Generate(ctx context.Context, prompt string) (string, error)
	// GenerateWithTools 生成文本，支持工具调用
	GenerateWithTools(ctx context.Context, prompt string, tools []*ToolRegistry, maxIterations int) (string, []ToolCall, error)
	// GenerateSubTasks 根据任务生成子任务计划
	GenerateSubTasks(ctx context.Context, taskName string, taskDesc string, depth int, maxDepth int) (*SubTaskPlan, error)
	// GetLastUsage 获取上一次调用的 Token 使用量
	GetLastUsage() Usage
	// Name 返回 Provider 名称
	Name() string
}

// LLMProviderFactory 基础设施层实现，用于创建实际的 LLMClient
type LLMProviderFactory interface {
	// Build 根据配置创建 LLMClient
	Build(config *LLMProviderConfig) (LLMClient, error)
}
