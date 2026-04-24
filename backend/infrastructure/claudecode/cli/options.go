package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/weibh/taskmanager/domain"
)

// buildCLIArgs 构建 claude CLI 参数
func buildCLIArgs(userInput, cliSessionID string, provider *domain.LLMProvider, config *domain.ClaudeCodeConfig) []string {
	args := []string{"-p", "--output-format=stream-json", "--verbose"}

	// 模型
	model := ""
	if config != nil && config.Model != "" {
		model = config.Model
	} else if provider != nil {
		model = provider.DefaultModel()
	}
	if model != "" {
		args = append(args, "--model", model)
	}

	// 系统提示词
	if config != nil && config.SystemPrompt != "" {
		args = append(args, "--system-prompt", config.SystemPrompt)
	}

	// 追加系统提示词
	if config != nil && config.AppendSystemPrompt != "" {
		args = append(args, "--append-system-prompt", config.AppendSystemPrompt)
	}

	// 最大思考 token
	if config != nil && config.MaxThinkingTokens > 0 {
		args = append(args, "--max-thinking-tokens", fmt.Sprintf("%d", config.MaxThinkingTokens))
	}

	// 权限模式
	if config != nil && config.PermissionMode == domain.PermissionModeBypassPermissions {
		args = append(args, "--dangerously-skip-permissions")
	} else if config != nil && config.PermissionMode != "" {
		args = append(args, "--permission-mode", string(config.PermissionMode))
	} else {
		args = append(args, "--permission-mode", "default")
	}

	// 允许的工具
	if config != nil && len(config.AllowedTools) > 0 {
		args = append(args, "--allowed-tools", strings.Join(config.AllowedTools, ","))
	}

	// 禁止的工具
	if config != nil && len(config.DisallowedTools) > 0 {
		args = append(args, "--disallowed-tools", strings.Join(config.DisallowedTools, ","))
	}

	// 工作目录
	if config != nil && config.Cwd != "" {
		args = append(args, "--add-dir", config.Cwd)
	}

	// 会话恢复
	if config != nil && config.Resume != nil && *config.Resume && cliSessionID != "" {
		args = append(args, "--resume", cliSessionID)
	}

	// 继续会话
	if config != nil && config.ContinueConversation != nil && *config.ContinueConversation {
		args = append(args, "-c")
	}

	// Fork 会话
	if config != nil && config.ForkSession != nil && *config.ForkSession {
		args = append(args, "--fork-session")
	}

	// 备用模型
	if config != nil && config.FallbackModel != "" {
		args = append(args, "--fallback-model", config.FallbackModel)
	}

	// 文件检查点
	if config != nil && config.FileCheckpointing != nil && *config.FileCheckpointing {
		args = append(args, "--file-checkpointing")
	}

	// JSON Schema
	if config != nil && len(config.JSONSchema) > 0 {
		data, _ := json.Marshal(config.JSONSchema)
		args = append(args, "--json-schema", string(data))
	}

	// 包含部分消息
	if config != nil && config.IncludePartialMessages != nil && *config.IncludePartialMessages {
		args = append(args, "--include-partial-messages")
	}

	// 最大预算
	if config != nil && config.MaxBudgetUSD > 0 {
		args = append(args, "--max-budget-usd", fmt.Sprintf("%f", config.MaxBudgetUSD))
	}

	// Beta 功能
	if config != nil && len(config.Betas) > 0 {
		args = append(args, "--betas")
		args = append(args, config.Betas...)
	}

	// MCP 配置
	if config != nil && len(config.McpServers) > 0 {
		for name, server := range config.McpServers {
			mcpData := map[string]any{
				"command": server.Command,
				"args":    server.Args,
				"env":     server.Env,
			}
			data, _ := json.Marshal(map[string]any{name: mcpData})
			args = append(args, "--mcp-config", string(data))
		}
	}

	// 额外参数
	if config != nil && len(config.ExtraArgs) > 0 {
		for key, val := range config.ExtraArgs {
			args = append(args, "--"+key, val)
		}
	}

	// 设置来源
	if config != nil && len(config.SettingSources) > 0 {
		args = append(args, "--setting-sources", strings.Join(config.SettingSources, ","))
	}

	// Session ID（如果未使用 --resume）
	if cliSessionID != "" && (config == nil || config.Resume == nil || !*config.Resume) {
		args = append(args, "--session-id", cliSessionID)
	}

	// 用户输入
	args = append(args, "--", userInput)

	return args
}

// buildEnv 构建环境变量
func buildEnv(provider *domain.LLMProvider, config *domain.ClaudeCodeConfig) []string {
	env := os.Environ()

	// 从 Provider 注入 API Key 和 Base URL
	if provider != nil {
		if apiKey := provider.APIKey(); apiKey != "" {
			env = append(env, "ANTHROPIC_AUTH_TOKEN="+apiKey)
		}
		if baseURL := provider.APIBase(); baseURL != "" {
			env = append(env, "ANTHROPIC_BASE_URL="+baseURL)
		}
	}

	// 从配置中合并环境变量
	if config != nil && len(config.Env) > 0 {
		for k, v := range config.Env {
			env = append(env, k+"="+v)
		}
	}

	return env
}

// ValidateClaudeAvailable 检查 claude CLI 是否可用
func ValidateClaudeAvailable() error {
	path, err := exec.LookPath("claude")
	if err != nil {
		return &ClaudeNotFoundError{err: err}
	}
	cmd := exec.Command(path, "--version")
	_, err = cmd.Output()
	if err != nil {
		return &ClaudeNotFoundError{err: err}
	}
	return nil
}

// ClaudeNotFoundError claude CLI 未找到错误
type ClaudeNotFoundError struct {
	err error
}

func (e *ClaudeNotFoundError) Error() string {
	if e.err != nil {
		return "claude CLI not found: " + e.err.Error()
	}
	return "claude CLI not found in PATH"
}
