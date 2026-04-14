package opencode

import (
	"os"
	"os/exec"
	"strings"

	"github.com/weibh/taskmanager/domain"
)

// buildCLIArgs 构建 OpenCode CLI 参数
func buildCLIArgs(userInput, workDir string, provider *domain.LLMProvider, config *domain.OpenCodeConfig, sessionID string) []string {
	args := []string{"run"}

	// 模型：仅当用户在 opencode_config 中显式配置时才传入
	// 不传 --model 时 opencode 会使用默认内置免费模型
	if config != nil && config.Model != "" {
		model := config.Model
		if provider != nil && !strings.Contains(model, "/") {
			providerType := provider.ProviderType()
			if providerType == "" {
				providerType = "anthropic"
			}
			model = providerType + "/" + model
		}
		args = append(args, "--model", model)
	}

	// 工作目录
	workDirToUse := config.WorkDir
	if workDirToUse == "" {
		workDirToUse = workDir
	}
	if workDirToUse != "" {
		args = append(args, "--dir", workDirToUse)
	}

	// 会话
	if sessionID != "" {
		args = append(args, "--session", sessionID)
	} else if config.Continue {
		args = append(args, "--continue")
	}

	// 分叉会话
	if config.Fork {
		args = append(args, "--fork")
	}

	// Agent 类型
	if config.AgentType != "" {
		args = append(args, "--agent", string(config.AgentType))
	}

	// 权限跳过
	if config.SkipPermissions {
		args = append(args, "--dangerously-skip-permissions")
	}

	// 思考过程
	if config.ShowThinking {
		args = append(args, "--thinking")
	}

	// 分享会话
	if config.ShareSession {
		args = append(args, "--share")
	}

	// 模型变体
	if config.Variant != "" {
		args = append(args, "--variant", config.Variant)
	}

	// 输出格式（固定为 JSON）
	args = append(args, "--format", "json")

	// 用户消息
	args = append(args, "--", userInput)

	return args
}

// buildEnv 构建环境变量
func buildEnv(provider *domain.LLMProvider, config *domain.OpenCodeConfig) []string {
	env := os.Environ()

	// 注意：不自动注入 Provider 的 API key/baseURL，避免干扰 opencode 使用默认内置免费模型。
	// 如需指定凭证或模型，请通过 config.Env 显式传入。

	// 从配置中合并环境变量
	if config != nil && len(config.Env) > 0 {
		for k, v := range config.Env {
			env = append(env, k+"="+v)
		}
	}

	// 系统提示词（通过环境变量传递）
	if config != nil && config.SystemPrompt != "" {
		env = append(env, "OPENCODE_SYSTEM_PROMPT="+config.SystemPrompt)
	}

	return env
}

// ValidateOpenCodeAvailable 检查 opencode 是否可用
func ValidateOpenCodeAvailable() error {
	path, err := exec.LookPath("opencode")
	if err != nil {
		return &OpenCodeNotFoundError{}
	}

	cmd := exec.Command(path, "--version")
	_, err = cmd.Output()
	if err != nil {
		return &OpenCodeNotFoundError{err: err}
	}

	return nil
}

// OpenCodeNotFoundError OpenCode 未找到错误
type OpenCodeNotFoundError struct {
	err error
}

func (e *OpenCodeNotFoundError) Error() string {
	if e.err != nil {
		return "opencode not found: " + e.err.Error()
	}
	return "opencode not found in PATH"
}
