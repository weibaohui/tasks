# Claude Code 配置项文档

## 概述

本文档列出 Claude Code 直接 CLI 调用支持的配置项（架构变更：PR #35 已移除 `claude-agent-sdk-go` SDK，改用直接调用 `claude` CLI）。

实现位置：`backend/infrastructure/claudecode/cli/`

---

## 1. 工具控制 (Tool Control)

### 1.1 AllowedTools - 允许的工具列表
- **类型**: `[]string`
- **说明**: 白名单，只有列表中的工具可以被调用
- **CLI 参数**: `--allowed-tools`
- **示例**: `WithAllowedTools("Read", "Write", "Bash")`

### 1.2 DisallowedTools - 禁止的工具列表
- **类型**: `[]string`
- **说明**: 黑名单，列表中的工具将被禁用
- **CLI 参数**: `--disallowed-tools`
- **示例**: `WithDisallowedTools("Bash", "Write")`

### 1.3 Tools - 可用工具
- **类型**: `[]string` 或 `ToolsPreset`
- **说明**: 直接设置可用的工具列表，或使用预设配置
- **CLI 参数**: `WithTools(tools ...string)` 或 `WithToolsPreset(preset string)`
- **预设值**: `"claude_code"` - 启用 Claude Code 预设工具
- **示例**: `WithTools("Read", "Write", "Edit", "Bash", "Glob", "Grep")`

---

## 2. 模型 & System Prompt

### 2.1 Model - 模型
- **类型**: `string`
- **说明**: 设置要使用的 AI 模型
- **CLI 参数**: `WithModel(model string)`
- **当前值**: `"MiniMax-M2.7-highspeed"`
- **示例**: `WithModel("MiniMax-M2.7-highspeed")`

### 2.2 FallbackModel - 备用模型
- **类型**: `string`
- **说明**: 当主模型不可用时使用的备用模型
- **CLI 参数**: `WithFallbackModel(model string)`

### 2.3 SystemPrompt - 系统提示词
- **类型**: `string`
- **说明**: 设置系统级提示词，定义 Agent 的行为和角色
- **CLI 参数**: `WithSystemPrompt(prompt string)`

### 2.4 AppendSystemPrompt - 追加系统提示词
- **类型**: `string`
- **说明**: 在现有系统提示词后追加内容
- **CLI 参数**: `WithAppendSystemPrompt(prompt string)`

### 2.5 MaxThinkingTokens - 最大思考 Token 数
- **类型**: `int`
- **说明**: 允许模型用于思考的最大 Token 数量
- **默认值**: `8000`
- **CLI 参数**: `WithMaxThinkingTokens(tokens int)`
- **示例**: `WithMaxThinkingTokens(10000)`

---

## 3. 权限 & 安全 (Permission & Safety)

### 3.1 PermissionMode - 权限模式
- **类型**: `PermissionMode` (枚举)
- **说明**: 控制工具调用的权限处理方式
- **CLI 参数**: `WithPermissionMode(mode PermissionMode)`
- **可选值**:
  - `PermissionModeDefault` - 标准权限处理（默认）
  - `PermissionModeAcceptEdits` - 自动接受所有编辑操作
  - `PermissionModePlan` - 计划模式，仅规划不执行
  - `PermissionModeBypassPermissions` - 绕过所有权限检查

### 3.2 CanUseTool - 权限回调
- **类型**: `CanUseToolCallback`
- **说明**: 动态控制工具使用权限的回调函数
- **CLI 参数**: `WithCanUseTool(callback CanUseToolCallback)`
- **返回**: `PermissionResultAllow` 或 `PermissionResultDeny`
- **使用场景**:
  - 基于内容过滤（如检查文件路径）
  - 审计日志记录
  - 动态权限决策

---

## 4. 会话管理 (Session Management)

### 4.1 Resume - 恢复会话
- **类型**: `string` (Session ID)
- **说明**: 通过 Session ID 恢复之前的会话上下文
- **CLI 参数**: `WithResume(sessionID string)`
- **用途**: 实现多轮对话支持

### 4.2 MaxTurns - 最大对话轮次
- **类型**: `int`
- **说明**: 限制会话中的最大交互次数
- **CLI 参数**: `WithMaxTurns(turns int)`
- **示例**: `WithMaxTurns(10)`

### 4.3 ContinueConversation - 继续会话
- **类型**: `bool`
- **说明**: 是否启用会话继续功能
- **CLI 参数**: `WithContinueConversation(continueConversation bool)`

### 4.4 ForkSession - Fork 会话
- **类型**: `bool`
- **说明**: 恢复会话时是否 fork 到新 Session ID
- **CLI 参数**: `WithForkSession(fork bool)`

---

## 5. 文件 & 上下文 (File & Context)

### 5.1 Cwd - 工作目录
- **类型**: `string`
- **说明**: 设置 Claude Code 执行的工作目录
- **CLI 参数**: `WithCwd(cwd string)`
- **示例**: `WithCwd("/path/to/project")`

### 5.2 AddDirs - 添加目录
- **类型**: `[]string`
- **说明**: 将指定目录添加到上下文
- **CLI 参数**: `WithAddDirs(dirs ...string)`

### 5.3 FileCheckpointing - 文件检查点
- **类型**: `bool`
- **说明**: 启用文件变更跟踪，支持回滚到之前状态
- **CLI 参数**: `WithFileCheckpointing()` 或 `WithEnableFileCheckpointing(enable bool)`
- **用途**: 撤销文件修改、文件历史管理

---

## 6. MCP 服务器 (MCP Integration)

### 6.1 McpServers - MCP 服务器配置
- **类型**: `map[string]McpServerConfig`
- **说明**: 配置 MCP 服务器以扩展工具能力
- **CLI 参数**: `WithMcpServers(servers map[string]McpServerConfig)`
- **服务器类型**:
  - `McpServerTypeStdio` - 标准输入/输出模式
  - `McpServerTypeSSE` - Server-Sent Events 模式
  - `McpServerTypeHTTP` - HTTP 模式
  - `McpServerTypeSdk` - SDK 内置模式

### 6.2 SdkMcpServer - SDK 内置 MCP 服务器
- **类型**: `McpSdkServerConfig`
- **说明**: 在进程中运行的 MCP 服务器
- **CLI 参数**: `WithSdkMcpServer(name string, server *McpSdkServerConfig)`

---

## 7. 沙箱安全 (Sandbox Security)

### 7.1 SandboxEnabled - 启用沙箱
- **类型**: `bool`
- **说明**: 启用/禁用 Bash 命令沙箱隔离
- **CLI 参数**: `WithSandboxEnabled(enabled bool)`
- **注意**: 仅支持 Linux 和 macOS

### 7.2 AutoAllowBashIfSandboxed - 沙箱模式下自动批准 Bash
- **类型**: `bool`
- **说明**: 沙箱模式下自动批准 Bash 命令执行
- **CLI 参数**: `WithAutoAllowBashIfSandboxed(autoAllow bool)`

### 7.3 ExcludedCommands - 排除命令
- **类型**: `[]string`
- **说明**: 始终绕过沙箱的命令列表
- **CLI 参数**: `WithSandboxExcludedCommands(commands ...string)`
- **示例**: `WithSandboxExcludedCommands("git", "docker", "npm")`

### 7.4 SandboxNetwork - 沙箱网络配置
- **类型**: `SandboxNetworkConfig`
- **说明**: 配置沙箱内的网络访问权限
- **CLI 参数**: `WithSandboxNetwork(network *SandboxNetworkConfig)`
- **子配置**:
  - `AllowUnixSockets` - 允许的 Unix Socket 路径
  - `AllowAllUnixSockets` - 允许所有 Unix Socket
  - `AllowLocalBinding` - 允许绑定本地端口
  - `HTTPProxyPort` - HTTP 代理端口
  - `SOCKSProxyPort` - SOCKS5 代理端口

### 7.5 IgnoreViolations - 忽略违规
- **类型**: `SandboxIgnoreViolations`
- **说明**: 指定忽略的沙箱违规模式
- **子配置**:
  - `File` - 忽略的文件路径模式
  - `Network` - 忽略的网络主机模式

---

## 8. 插件 (Plugins)

### 8.1 Plugins - 插件列表
- **类型**: `[]SdkPluginConfig`
- **说明**: 配置要加载的插件
- **CLI 参数**: `WithPlugins(plugins []SdkPluginConfig)`

### 8.2 LocalPlugin - 本地插件
- **类型**: `string` (路径)
- **说明**: 从本地路径加载插件
- **CLI 参数**: `WithLocalPlugin(path string)`

---

## 9. 结构化输出 (Structured Output)

### 9.1 JSONSchema - JSON Schema 约束
- **类型**: `map[string]any`
- **说明**: 约束 Claude 响应符合指定 JSON Schema
- **CLI 参数**: `WithJSONSchema(schema map[string]any)`
- **用途**: 提取结构化数据、类型安全响应

### 9.2 OutputFormat - 输出格式
- **类型**: `OutputFormat`
- **说明**: 设置响应输出格式
- **CLI 参数**: `WithOutputFormat(format *OutputFormat)`

---

## 10. 生命周期钩子 (Hooks)

### 10.1 PreToolUse - 工具执行前钩子
- **触发时机**: 工具执行前
- **用途**: 阻止危险命令、修改输入参数、记录日志
- **CLI 参数**: `WithPreToolUseHook(matcher string, callback HookCallback)`

### 10.2 PostToolUse - 工具执行后钩子
- **触发时机**: 工具执行后
- **用途**: 添加上下文、修改输出、审计追踪
- **CLI 参数**: `WithPostToolUseHook(matcher string, callback HookCallback)`

### 10.3 UserPromptSubmit - 用户提交提示时
- **触发时机**: 用户提交提示词时
- **用途**: 提示词验证、上下文注入
- **CLI 参数**: `WithHook(HookEventUserPromptSubmit, matcher, callback)`

### 10.4 Stop - 会话停止时
- **触发时机**: 会话结束前
- **用途**: 资源清理、最终状态保存
- **CLI 参数**: `WithHook(HookEventStop, matcher, callback)`

### 10.5 SubagentStop - 子代理停止时
- **触发时机**: 子代理会话结束时
- **CLI 参数**: `WithHook(HookEventSubagentStop, matcher, callback)`

### 10.6 PreCompact - 上下文压缩前
- **触发时机**: 上下文压缩/精简前
- **用途**: 选择性保留信息
- **CLI 参数**: `WithHook(HookEventPreCompact, matcher, callback)`

---

## 11. Beta 功能

### 11.1 Betas - Beta 特性列表
- **类型**: `[]SdkBeta`
- **说明**: 启用实验性功能
- **CLI 参数**: `WithBetas(betas ...SdkBeta)`
- **可用值**:
  - `SdkBetaContext1M` - 启用 1M 上下文窗口 (`"context-1m-2025-08-07"`)

---

## 12. 其他配置

### 12.1 MaxBudgetUSD - 最大预算
- **类型**: `float64`
- **说明**: API 使用的最大预算 (USD)
- **CLI 参数**: `WithMaxBudgetUSD(budget float64)`

### 12.2 IncludePartialMessages - 部分消息流
- **类型**: `bool`
- **说明**: 启用流式部分消息更新
- **CLI 参数**: `WithIncludePartialMessages(include bool)`

### 12.3 DebugWriter - 调试输出
- **类型**: `io.Writer`
- **说明**: CLI 调试输出目标
- **CLI 参数**: `WithDebugWriter(w io.Writer)`
- **常用值**: `os.Stderr`, `io.Discard`

### 12.4 StderrCallback - 标准错误回调
- **类型**: `func(string)`
- **说明**: 接收 CLI stderr 输出
- **CLI 参数**: `WithStderrCallback(callback func(string))`

### 12.5 CLIPath - CLI 路径
- **类型**: `string`
- **说明**: 自定义 Claude Code CLI 路径
- **CLI 参数**: `WithCLIPath(path string)`

### 12.6 Env - 环境变量
- **类型**: `map[string]string`
- **说明**: 为子进程设置环境变量
- **CLI 参数**: `WithEnv(env map[string]string)`
- **示例**: `WithEnv(map[string]string{"ANTHROPIC_API_KEY": "sk-xxx"})`

### 12.7 ExtraArgs - 额外 CLI 参数
- **类型**: `map[string]*string`
- **说明**: 传递额外 CLI 参数
- **CLI 参数**: `WithExtraArgs(args map[string]*string)`

### 12.8 Settings - 设置文件
- **类型**: `string`
- **说明**: 设置文件路径或 JSON 字符串
- **CLI 参数**: `WithSettings(settings string)`

### 12.9 SettingSources - 设置来源
- **类型**: `[]SettingSource`
- **说明**: 指定加载哪些设置源
- **CLI 参数**: `WithSettingSources(sources ...SettingSource)`
- **可用值**: `SettingSourceUser`, `SettingSourceProject`, `SettingSourceLocal`

---

## Agent 编辑界面配置分组

### Tab 1: 基本设置

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| **Model** | 模型 | MiniMax-M2.7-highspeed |
| **SystemPrompt** | 系统提示词 | - |
| **MaxThinkingTokens** | 最大思考 Token 数 | 8000 |
| **PermissionMode** | 权限模式 | default |
| **AllowedTools** | 允许的工具 | 全部 |
| **DisallowedTools** | 禁止的工具 | 空 |
| **MaxTurns** | 最大对话轮次 | 无限制 |
| **Cwd** | 工作目录 | 当前目录 |
| **Resume** | 恢复会话 | 关闭 |

---

### Tab 2: 高级设置

#### 模型 & Prompt

| 配置项 | 说明 |
|--------|------|
| FallbackModel | 备用模型 |
| AppendSystemPrompt | 追加系统提示词 |

#### 会话 & 文件

| 配置项 | 说明 |
|--------|------|
| FileCheckpointing | 启用文件检查点 |
| ContinueConversation | 继续会话 |
| ForkSession | Fork 会话 |

#### 沙箱安全

| 配置项 | 说明 |
|--------|------|
| SandboxEnabled | 启用沙箱 |
| AutoAllowBashIfSandboxed | 沙箱自动批准 Bash |
| ExcludedCommands | 沙箱排除命令 |
| SandboxNetwork | 网络配置 |

#### MCP & 插件

| 配置项 | 说明 |
|--------|------|
| McpServers | MCP 服务器列表 |
| Plugins | 插件列表 |
| LocalPlugin | 本地插件路径 |

#### 输出 & 调试

| 配置项 | 说明 |
|--------|------|
| JSONSchema | JSON Schema 约束 |
| IncludePartialMessages | 流式部分消息 |
| MaxBudgetUSD | 最大预算 (USD) |
| DebugWriter | 调试输出目标 |
| StderrCallback | Stderr 回调 |

#### 其他

| 配置项 | 说明 |
|--------|------|
| Betas | Beta 功能 |
| CLIPath | CLI 路径 |
| Env | 环境变量 |
| ExtraArgs | 额外 CLI 参数 |
| Settings | 设置文件 |
| SettingSources | 设置来源 |
| CanUseTool | 权限回调函数 |
