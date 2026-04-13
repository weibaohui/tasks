package claudecode

import (
	claudecode "github.com/severity1/claude-agent-sdk-go"
	"github.com/weibh/taskmanager/domain"
	"go.uber.org/zap"
)

func (p *ClaudeCodeProcessor) buildOptions(provider *domain.LLMProvider, cliSessionID string, agent *domain.Agent, toolHookAdapter *toolHookAdapter) []claudecode.Option {
	opts := []claudecode.Option{}

	// 获取配置
	var config *domain.ClaudeCodeConfig
	if agent != nil {
		config = agent.ClaudeCodeConfig()
	}
	if config == nil {
		config = domain.DefaultClaudeCodeConfig()
	}

	// 注册工具钩子以记录对话
	if toolHookAdapter != nil {
		opts = append(opts, claudecode.WithPreToolUseHook("", toolHookAdapter.preToolUseAdapter))
		opts = append(opts, claudecode.WithPostToolUseHook("", toolHookAdapter.postToolUseAdapter))
	}

	// 设置 Env（API Key 和 Base URL）
	env := config.Env
	if env == nil {
		env = make(map[string]string)
	}

	// 设置模型
	model := config.Model
	if model == "" && agent != nil {
		model = agent.Model()
	}
	if model == "" {
		if provider != nil {
			// 当模型为空时，从 provider 获取 API Key 和 Base URL
			if provider.APIKey() != "" {
				env["ANTHROPIC_API_KEY"] = provider.APIKey()
			}
			if provider.APIBase() != "" {
				env["ANTHROPIC_BASE_URL"] = provider.APIBase()
			}
			// 模型保持为空，让 Claude Code 使用默认模型
		} else {
			// 没有 provider 时，使用默认模型
			model = "MiniMax-M2.7-highspeed"
		}
	}

	opts = append(opts, claudecode.WithEnv(env))
	opts = append(opts, claudecode.WithModel(model))

	// 设置系统提示词
	if config.SystemPrompt != "" {
		opts = append(opts, claudecode.WithSystemPrompt(config.SystemPrompt))
	}

	// 设置最大思考 Token
	if config.MaxThinkingTokens > 0 {
		opts = append(opts, claudecode.WithMaxThinkingTokens(config.MaxThinkingTokens))
	}

	// 设置权限模式
	switch config.PermissionMode {
	case domain.PermissionModeAcceptEdits:
		opts = append(opts, claudecode.WithPermissionMode(claudecode.PermissionModeAcceptEdits))
	case domain.PermissionModePlan:
		opts = append(opts, claudecode.WithPermissionMode(claudecode.PermissionModePlan))
	case domain.PermissionModeBypassPermissions:
		opts = append(opts, claudecode.WithPermissionMode(claudecode.PermissionModeBypassPermissions))
	default:
		opts = append(opts, claudecode.WithPermissionMode(claudecode.PermissionModeDefault))
	}

	// 设置允许的工具
	if len(config.AllowedTools) > 0 {
		opts = append(opts, claudecode.WithAllowedTools(config.AllowedTools...))
	}

	// 设置禁止的工具
	if len(config.DisallowedTools) > 0 {
		opts = append(opts, claudecode.WithDisallowedTools(config.DisallowedTools...))
	}

	// 设置最大对话轮次
	if config.MaxTurns > 0 {
		opts = append(opts, claudecode.WithMaxTurns(config.MaxTurns))
	}

	// 设置工作目录
	if config.Cwd != "" {
		opts = append(opts, claudecode.WithCwd(config.Cwd))
	}

	// 设置会话恢复
	if config.Resume != nil && *config.Resume && cliSessionID != "" {
		opts = append(opts, claudecode.WithResume(cliSessionID))
	}

	// 设置继续会话
	if config.ContinueConversation != nil && *config.ContinueConversation {
		opts = append(opts, claudecode.WithContinueConversation(true))
	}

	// 设置文件检查点
	if config.FileCheckpointing != nil && *config.FileCheckpointing {
		opts = append(opts, claudecode.WithFileCheckpointing())
	}

	// 设置备用模型
	if config.FallbackModel != "" {
		opts = append(opts, claudecode.WithFallbackModel(config.FallbackModel))
	}

	// 追加系统提示词
	if config.AppendSystemPrompt != "" {
		opts = append(opts, claudecode.WithAppendSystemPrompt(config.AppendSystemPrompt))
	}

	// 设置沙箱
	if config.SandboxEnabled != nil && *config.SandboxEnabled {
		opts = append(opts, claudecode.WithSandboxEnabled(true))
		if config.AutoAllowBashIfSandboxed != nil && *config.AutoAllowBashIfSandboxed {
			opts = append(opts, claudecode.WithAutoAllowBashIfSandboxed(true))
		}
		if len(config.ExcludedCommands) > 0 {
			opts = append(opts, claudecode.WithSandboxExcludedCommands(config.ExcludedCommands...))
		}
	}

	// 设置 MCP 服务器
	if len(config.McpServers) > 0 {
		mcpServers := make(map[string]claudecode.McpServerConfig)
		for name, server := range config.McpServers {
			// 只支持 stdio 类型
			mcpServers[name] = &claudecode.McpStdioServerConfig{
				Command: server.Command,
				Args:    server.Args,
				Env:     server.Env,
			}
		}
		opts = append(opts, claudecode.WithMcpServers(mcpServers))
	}

	// 设置插件
	if len(config.Plugins) > 0 {
		// 需要将 string 转换为 SdkPluginConfig
		// 这里暂时跳过，实际使用时需要根据 SDK 定义转换
	}

	// 设置本地插件
	if config.LocalPlugin != "" {
		opts = append(opts, claudecode.WithLocalPlugin(config.LocalPlugin))
	}

	// 设置 JSON Schema
	if len(config.JSONSchema) > 0 {
		opts = append(opts, claudecode.WithJSONSchema(config.JSONSchema))
	}

	// 设置部分消息
	if config.IncludePartialMessages != nil && *config.IncludePartialMessages {
		opts = append(opts, claudecode.WithIncludePartialMessages(true))
	}

	// 设置最大预算
	if config.MaxBudgetUSD > 0 {
		opts = append(opts, claudecode.WithMaxBudgetUSD(config.MaxBudgetUSD))
	}

	// 设置 Beta 功能
	if len(config.Betas) > 0 {
		// 需要将 string 转换为 SdkBeta
		// 这里暂时跳过，实际使用时需要根据 SDK 定义转换
	}

	// 设置 CLI 路径
	if config.CLIPath != "" {
		opts = append(opts, claudecode.WithCLIPath(config.CLIPath))
	}

	// 设置额外参数
	if len(config.ExtraArgs) > 0 {
		args := make(map[string]*string)
		for k, v := range config.ExtraArgs {
			args[k] = &v
		}
		opts = append(opts, claudecode.WithExtraArgs(args))
	}

	// 设置来源
	if len(config.SettingSources) > 0 {
		// 需要将 string 转换为 SettingSource
		// 这里暂时跳过，实际使用时需要根据 SDK 定义转换
	}

	p.logger.Info("Claude Code 选项配置",
		zap.String("provider", func() string {
			if provider != nil {
				return provider.ProviderKey()
			}
			return "default"
		}()),
		zap.String("api_base_url", func() string {
			if provider != nil {
				return provider.APIBase()
			}
			return ""
		}()),
		zap.String("cli_session_id", cliSessionID),
		zap.String("model", model),
		zap.Int("options_count", len(opts)),
	)

	return opts
}
