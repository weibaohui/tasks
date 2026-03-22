/**
 * LLM Provider 接口定义
 * 支持多种 LLM 实现：OpenAI、Claude、Ollama 等
 */
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// SubTask 子任务结构
type SubTask struct {
	Goal     string `json:"goal"`
	TaskType string `json:"type"` // data_processing, file_operation, api_call, agent, custom
}

// SubTaskPlan 子任务计划
type SubTaskPlan struct {
	SubTasks []SubTask `json:"sub_tasks"`
	Reason   string    `json:"reason,omitempty"`
}

// LLMProvider LLM provider 接口
type LLMProvider interface {
	// Generate 生成文本
	Generate(ctx context.Context, prompt string) (string, error)

	// GenerateSubTasks 根据任务生成子任务计划
	GenerateSubTasks(ctx context.Context, taskName string, taskDesc string, depth int, maxDepth int) (*SubTaskPlan, error)

	// Name 返回 provider 名称
	Name() string
}

// Config LLM 配置
type Config struct {
	ProviderType string // "openai", "claude", "ollama"
	Model        string // 模型名称
	APIKey       string
	BaseURL      string // 可选的自定义端点
	Temperature  float64
	MaxTokens    int
}

// DefaultConfig 从环境变量创建默认配置
func DefaultConfig() *Config {
	providerType := os.Getenv("LLM_PROVIDER")
	if providerType == "" {
		providerType = "openai" // 默认使用 OpenAI
	}

	config := &Config{
		ProviderType: providerType,
		Model:        getEnvOrDefault("LLM_MODEL", "gpt-4"),
		APIKey:       os.Getenv("OPENAI_API_KEY"),
		BaseURL:      os.Getenv("LLM_BASE_URL"),
		Temperature:  0.7,
		MaxTokens:    4096,
	}

	// Claude 配置
	if providerType == "claude" {
		config.APIKey = os.Getenv("ANTHROPIC_API_KEY")
		if config.Model == "" {
			config.Model = "claude-3-opus-20240229"
		}
	}

	// Ollama 配置
	if providerType == "ollama" {
		config.BaseURL = getEnvOrDefault("OLLAMA_BASE_URL", "http://localhost:11434")
		config.Model = getEnvOrDefault("OLLAMA_MODEL", "llama2")
		config.APIKey = "" // Ollama 不需要 API key
	}

	return config
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// NewLLMProvider 创建 LLM Provider
func NewLLMProvider(config *Config) (LLMProvider, error) {
	switch config.ProviderType {
	case "openai":
		return NewOpenAIProvider(config)
	case "claude":
		return NewClaudeProvider(config)
	case "ollama":
		return NewOllamaProvider(config)
	default:
		return nil, fmt.Errorf("unknown provider type: %s", config.ProviderType)
	}
}

// subTaskPrompt 生成子任务的 prompt
func subTaskPrompt(taskName, taskDesc string, depth, maxDepth int) string {
	return fmt.Sprintf(`你是一个任务规划 Agent。请根据以下任务生成子任务计划。

任务信息：
- 任务名称：%s
- 任务描述：%s
- 当前深度：%d/%d

请生成 1-3 个子任务，每个子任务包含：
- goal: 任务目标描述
- type: 任务类型（data_processing, file_operation, api_call, agent, custom 之一）

注意：
1. 子任务应该是完成父任务所需的关键步骤
2. 任务类型应与子任务内容匹配
3. 如果当前深度已达到最大深度，则不生成子任务

请以 JSON 格式返回：
{
  "sub_tasks": [
    {"goal": "处理前50%数据", "type": "data_processing"},
    {"goal": "处理后50%数据", "type": "file_operation"}
  ],
  "reason": "简要说明为什么这样分解"
}`, taskName, taskDesc, depth, maxDepth)
}

// extractJSON 从响应中提取 JSON
func extractJSON(s string) string {
	// 查找 ```json ... ``` 块
	start := 0
	end := len(s)

	// 查找 JSON 开始标记
	if idx := strings.Index(s, "```json"); idx >= 0 {
		start = idx + 7
	} else if idx := strings.Index(s, "```"); idx >= 0 {
		start = idx + 3
	}

	// 查找 JSON 结束标记
	if idx := strings.Index(s[start:], "```"); idx >= 0 {
		end = start + idx
	}

	return strings.TrimSpace(s[start:end])
}

// tryFixAndParseJSON 尝试修复并解析 JSON
func tryFixAndParseJSON(s string) (*SubTaskPlan, error) {
	var plan SubTaskPlan

	// 移除可能的前缀文本
	jsonStart := strings.Index(s, "{")
	if jsonStart < 0 {
		return nil, fmt.Errorf("no JSON object found")
	}

	jsonStr := s[jsonStart:]

	// 尝试解析
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		// 最后尝试：替换常见的问题字符
		fixed := fixJSON(jsonStr)
		if err := json.Unmarshal([]byte(fixed), &plan); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	}

	return &plan, nil
}

// fixJSON 修复常见的 JSON 问题
func fixJSON(s string) string {
	// 移除单引号替换为双引号（如果有问题）
	// 这是一个简单的修复，不处理所有情况
	return s
}
