/**
 * LLM Provider 接口定义
 * 支持多种 LLM 实现：OpenAI、Claude、Ollama 等
 */
package llm

import (
	"context"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// SubTask 子任务结构
type SubTask struct {
	Goal     string `yaml:"goal"`
	TaskType string `yaml:"type"` // agent, coding, custom
}

// SubTaskPlan 子任务计划
type SubTaskPlan struct {
	SubTasks []SubTask `yaml:"sub_tasks"`
	Reason   string    `yaml:"reason,omitempty"`
}

// LLMProvider LLM provider 接口
type LLMProvider interface {
	// Generate 生成文本
	Generate(ctx context.Context, prompt string) (string, error)

	// GenerateWithTools 生成文本，支持工具调用
	// tools: 可用的工具列表
	// maxIterations: 最大工具调用迭代次数
	GenerateWithTools(ctx context.Context, prompt string, tools []*ToolRegistry, maxIterations int) (string, []ToolCall, error)

	// GenerateSubTasks 根据任务生成子任务计划
	GenerateSubTasks(ctx context.Context, taskName string, taskDesc string, depth int, maxDepth int) (*SubTaskPlan, error)

	// GetLastUsage 返回上次调用的 token 使用量
	GetLastUsage() Usage

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

// NewLLMProvider 创建 LLM Provider
func NewLLMProvider(config *Config) (LLMProvider, error) {
	switch config.ProviderType {
	case "openai", "minimax":
		// minimax 使用 OpenAI 兼容 API，使用 eino 实现
		return NewEinoProvider(config, nil)
	case "claude":
		// Claude 使用自定义实现
		return NewClaudeProvider(config)
	case "ollama":
		// Ollama 使用自定义实现
		return NewOllamaProvider(config)
	case "eino":
		// 显式使用 eino
		return NewEinoProvider(config, nil)
	default:
		return nil, fmt.Errorf("unknown provider type: %s", config.ProviderType)
	}
}

// subTaskPrompt 生成子任务的 prompt
// SubTaskPrompt 生成子任务的 prompt
func SubTaskPrompt(taskName, taskDesc string, depth, maxDepth int) string {
	return fmt.Sprintf(`你是一个任务规划 Agent。请根据以下任务生成子任务计划。

任务信息：
- 任务名称：%s
- 任务描述：%s
- 当前深度：%d/%d

【重要】子任务创建原则：
1. 只有在确实有必要时才创建子任务（如：任务需要多人协作、任务包含多个独立可并行部分、任务需要不同技能）
2. 如果本身就能独立完成的任务，不要创建子任务
3. 避免无限向下创建子任务，优先尝试直接完成
4. 子任务应该是完成父任务的关键步骤，不是简单拆分
5. 任务类型应与子任务内容匹配
6. 如果任务简单或已达到最大深度，可以不分解（返回空 sub_tasks）
7. 子任务数量根据实际需求决定，不要强行分解

【关键】如果不需要分解子任务，reason 字段必须精确填写任务执行结果（如计算题写"49"，查找题写"找到的答案"），不要有任何额外解释、感叹、问候语或无关内容。

请直接返回 YAML 格式，不要包含任何解释或标记：
sub_tasks:
  - goal: 任务目标描述
    type: 任务类型
reason: 只写任务执行结果，不要写任何其他内容`, taskName, taskDesc, depth, maxDepth)
}

// extractYAML 从响应中提取 YAML
// ExtractYAML 从响应中提取 YAML
func ExtractYAML(s string) string {
	// 去除可能的 markdown 标记
	s = strings.TrimSpace(s)

	// 去除 ```yaml ``` 或 ``` 包裹
	if idx := strings.Index(s, "```yaml"); idx >= 0 {
		start := idx + 7
		if endIdx := strings.Index(s[start:], "```"); endIdx >= 0 {
			return strings.TrimSpace(s[start : start+endIdx])
		}
	} else if idx := strings.Index(s, "```"); idx >= 0 {
		start := idx + 3
		if endIdx := strings.Index(s[start:], "```"); endIdx >= 0 {
			return strings.TrimSpace(s[start : start+endIdx])
		}
	}

	return s
}

// tryParseYAML 尝试解析 YAML
// TryParseYAML 尝试解析 YAML
func TryParseYAML(s string) (*SubTaskPlan, error) {
	var plan SubTaskPlan

	// 移除可能的前缀文本，找到 YAML 开始位置
	yamlStart := strings.Index(s, "sub_tasks:")
	if yamlStart < 0 {
		yamlStart = strings.Index(s, "sub_tasks :")
	}
	if yamlStart < 0 {
		// 尝试找其他开始标记
		for _, prefix := range []string{"- goal:", "reason:", "goal:", "type:"} {
			if idx := strings.Index(s, prefix); idx >= 0 && (yamlStart < 0 || idx < yamlStart) {
				yamlStart = idx
			}
		}
	}

	if yamlStart < 0 {
		return nil, fmt.Errorf("no YAML content found")
	}

	yamlStr := strings.TrimSpace(s[yamlStart:])

	if err := yaml.Unmarshal([]byte(yamlStr), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &plan, nil
}
