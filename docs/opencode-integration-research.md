# OpenCode CLI 集成可行性调研

## 概述

本文档调研通过 CLI 封装方式实现 OpenCode 集成的可行性，参考现有 Claude Code SDK 集成的架构设计。

---

## 一、OpenCode CLI 基础信息

### 1.1 安装方式

```bash
npm i -g opencode-ai@latest
# 或
brew install anomalyco/tap/opencode
```

当前版本：`1.4.3`

### 1.2 核心命令

| 命令 | 功能 |
|------|------|
| `opencode run [message]` | 单次任务执行（无需 PTY） |
| `opencode` | 启动交互式 TUI（需 PTY） |
| `opencode -c` | 继续上次会话 |
| `opencode -s <session_id>` | 继续指定会话 |
| `opencode session list` | 列出所有会话 |
| `opencode session delete <id>` | 删除会话 |
| `opencode providers` | 管理 AI Provider |
| `opencode models` | 列出可用模型 |
| `opencode agent` | 管理 Agent |
| `opencode mcp` | 管理 MCP 服务器 |
| `opencode pr <number>` | PR 审查 |
| `opencode stats` | 用量统计 |
| `opencode serve` | 启动无头服务器 |
| `opencode acp` | 启动 ACP 服务器 |

### 1.3 认证配置

OpenCode 支持通过环境变量配置认证，**无需登录**：

```bash
ANTHROPIC_AUTH_TOKEN=sk-xxx \
ANTHROPIC_BASE_URL=https://api.anthropic.com \
opencode run "hello"
```

支持的认证方式：
- 环境变量：`ANTHROPIC_AUTH_TOKEN`、`ANTHROPIC_BASE_URL`
- Provider 登录：`opencode providers login`
- 配置文件：`~/.local/share/opencode/auth.json`

### 1.4 调试命令

```bash
opencode debug config        # 显示解析后的配置
opencode debug paths         # 显示全局路径
opencode debug skill         # 列出所有可用技能
opencode debug agent <name>  # 显示 Agent 配置详情
```

---

## 二、OpenCode CLI 配置项详解

### 2.1 全局选项（适用于所有命令）

| 选项 | 说明 | 默认值 |
|------|------|--------|
| `--print-logs` | 输出日志到 stderr | false |
| `--log-level` | 日志级别 | INFO |
| `--pure` | 不加载外部插件 | false |
| `--port` | 监听端口 | 0（随机） |
| `--hostname` | 监听主机名 | 127.0.0.1 |
| `--mdns` | 启用 mDNS 服务发现 | false |
| `--mdns-domain` | mDNS 域名 | opencode.local |
| `--cors` | CORS 额外域名 | [] |

### 2.2 `opencode run` 命令选项

| 选项 | 缩写 | 说明 | 示例 |
|------|------|------|------|
| `--command` | - | 要执行的命令（用于命令注入） | `--command "ls -la"` |
| `--continue` | `-c` | 继续上次会话 | `-c` |
| `--session` | `-s` | 继续指定会话 | `-s ses_xxx` |
| `--fork` | - | 分叉会话（与 -c 或 -s 配合） | `--fork` |
| `--share` | - | 分享会话，获取分享链接 | `--share` |
| `--model` | `-m` | 指定模型 | `-m openrouter/anthropic/claude-sonnet-4` |
| `--agent` | - | 指定 Agent 类型 | `--agent build` |
| `--format` | - | 输出格式：default/json | `--format json` |
| `--file` | `-f` | 附加文件到消息 | `-f config.yaml -f .env` |
| `--title` | - | 会话标题 | `--title "API实现"` |
| `--attach` | - | 附加到运行中的服务器 | `--attach http://localhost:4096` |
| `--password` | `-p` | 服务器认证密码 | `-p mypassword` |
| `--dir` | - | 工作目录 | `--dir /path/to/project` |
| `--variant` | - | 模型变体（推理努力） | `--variant high` |
| `--thinking` | - | 显示思考过程 | `--thinking` |
| `--dangerously-skip-permissions` | - | 自动批准所有权限 | `--dangerously-skip-permissions` |

### 2.3 Agent 类型详解

OpenCode 内置多种 Agent，分为 **primary**（主 Agent）和 **subagent**（子 Agent）：

| Agent | 类型 | 说明 | 主要用途 |
|--------|------|------|----------|
| `build` | primary | 全功能构建 Agent | 代码实现、修改 |
| `compaction` | primary | 精简版 Agent | 轻量级任务 |
| `plan` | primary | 规划 Agent | 只读 + 计划文件编辑 |
| `explore` | subagent | 探索 Agent | 代码分析、搜索 |
| `general` | subagent | 通用 Agent | 通用任务 |
| `summary` | primary | 摘要 Agent | 生成摘要 |
| `title` | primary | 标题 Agent | 生成标题 |

### 2.4 Agent 权限配置

每个 Agent 有独立的**基于 Pattern 的权限配置**，实现细粒度操作控制：

```json
{
  "permission": "bash",
  "action": "allow",    // allow | ask | deny
  "pattern": "*"        // glob pattern
}
```

**权限动作：**
- `allow` - 允许操作
- `ask` - 需要确认
- `deny` - 拒绝操作

**权限类型：**
- `read` - 读文件
- `edit` - 编辑文件
- `write` - 写文件
- `bash` - 执行命令
- `glob` - 文件匹配
- `grep` - 文本搜索
- `webfetch` - 获取网页
- `websearch` - 网页搜索
- `codesearch` - 代码搜索
- `todowrite` - 写待办事项
- `question` - 提问
- `doom_loop` - 死循环检测
- `plan_enter` - 进入规划模式
- `plan_exit` - 退出规划模式
- `external_directory` - 外部目录访问

### 2.5 Provider 管理

```bash
opencode providers list                    # 列出所有 Provider
opencode providers login [url]             # 登录 Provider（交互式）
opencode providers login -p <provider>     # 指定 Provider 登录
opencode providers logout                  # 登出
```

**支持的 Provider：**
- OpenRouter
- Anthropic
- OpenAI
- 自定义（通过 URL）

### 2.6 MCP 服务器管理

```bash
opencode mcp list                          # 列出 MCP 服务器
opencode mcp add                          # 添加 MCP 服务器
opencode mcp auth <name>                  # MCP OAuth 认证
opencode mcp logout <name>                # 移除 MCP 凭证
opencode mcp debug <name>                 # 调试 MCP 连接
```

### 2.7 其他命令

| 命令 | 功能 |
|------|------|
| `opencode session list` | 列出所有会话 |
| `opencode session delete <id>` | 删除会话 |
| `opencode stats` | 用量统计（总览） |
| `opencode stats --days 7` | 7 天用量统计 |
| `opencode stats --days 7 --models claude-sonnet-4` | 按模型统计 |
| `opencode models [provider]` | 列出可用模型 |
| `opencode pr <number>` | 获取并检出 PR 分支 |
| `opencode export [sessionID]` | 导出会话为 JSON |
| `opencode import <file>` | 从 JSON 导入会话 |
| `opencode plugin <module>` | 安装插件 |
| `opencode acp` | 启动 ACP 服务器 |
| `opencode serve` | 启动无头服务器 |
| `opencode web` | 启动 Web 界面 |

---

## 三、JSON 流式输出格式

### 3.1 格式概述

OpenCode CLI 通过 `--format json` 参数输出 JSON Lines（每行一个 JSON 对象），支持流式解析。

### 3.2 事件类型

| 事件类型 | 说明 |
|---------|------|
| `step_start` | 开始一个处理步骤 |
| `text` | 文本输出（模型回复） |
| `thinking` | 思考过程 |
| `tool_use` | 工具调用 |
| `step_finish` | 步骤结束 |

### 3.3 事件结构示例

**文本输出：**
```json
{
  "type": "text",
  "part": {
    "id": "prt_xxx",
    "messageID": "msg_xxx",
    "sessionID": "ses_xxx",
    "type": "text",
    "text": "Hello world",
    "time": {"start": 1234567890, "end": 1234567891}
  }
}
```

**思考过程：**
```json
{
  "type": "thinking",
  "part": {
    "thinking": "Let me analyze this...",
    "time": {"start": 1234567890, "end": 1234567891}
  }
}
```

**工具调用：**
```json
{
  "type": "tool_use",
  "part": {
    "type": "tool",
    "tool": "bash",
    "callID": "call_function_xxx",
    "state": {
      "status": "completed",  // completed | error | pending
      "input": {
        "command": "ls -la",
        "description": "List files"
      },
      "output": "total 0...",
      "metadata": {
        "exit": 0,
        "truncated": false
      }
    },
    "title": "List files",
    "time": {"start": 1234567890, "end": 1234567891}
  }
}
```

**步骤结束：**
```json
{
  "type": "step_finish",
  "part": {
    "reason": "stop",  // stop | tool-calls
    "tokens": {
      "total": 12345,
      "input": 100,
      "output": 200,
      "reasoning": 50,
      "cache": {"write": 1000, "read": 5000}
    },
    "cost": 0
  }
}
```

### 3.4 可用工具

根据测试，以下工具可用：
- `bash` - 执行 Shell 命令
- `write` - 写入文件
- `read` - 读取文件
- `edit` - 编辑文件
- `glob` - 文件匹配
- `grep` - 文本搜索
- `websearch` - 网页搜索
- `webfetch` - 获取网页内容

---

## 四、OpenCode vs Claude Code 功能对比

### 4.1 功能矩阵

| 功能 | Claude Code SDK | OpenCode CLI | 备注 |
|------|----------------|--------------|------|
| 单次任务执行 | ✅ | ✅ | `opencode run` |
| 流式文本输出 | ✅ | ✅ | JSON `text` 事件 |
| 思考过程 | ✅ | ✅ | JSON `thinking` 事件 |
| 工具调用拦截 | ✅ | ✅ | `tool_use` 事件 |
| Session 管理 | ✅ | ✅ | `--session` / `-c` |
| 继续会话 | ✅ | ✅ | `--continue` |
| 模型选择 | ✅ | ✅ | `--model` |
| 系统提示词 | ✅ | ⚠️ | 环境变量或配置文件 |
| 工具过滤 | ✅ | ⚠️ | Agent 类型 + Pattern |
| 工作目录 | ✅ | ✅ | `--dir` |
| MCP 服务器 | ✅ | ✅ | `opencode mcp` |
| 超时控制 | ✅ | ⚠️ | exec.CommandContext |
| 会话分叉 | ❌ | ✅ | `--fork` |
| 权限跳过 | ✅ | ✅ | `--dangerously-skip-permissions` |
| PR 审查 | ❌ | ✅ | `opencode pr <number>` |
| 用量统计 | ❌ | ✅ | `opencode stats` |
| Agent 类型 | ❌ | ✅ | `--agent` |

### 4.2 权限控制对比

**Claude Code:**
```go
claudecode.WithPermissionMode(claudecode.PermissionModeAcceptEdits)
```

**OpenCode:**
```bash
# 方式1: 跳过所有权限确认
opencode run '...' --dangerously-skip-permissions

# 方式2: 选择受限的 Agent
opencode run '...' --agent plan    # 只读 + 计划文件
opencode run '...' --agent explore  # 只读探索
```

### 4.3 模型选择对比

**Claude Code:**
```go
claudecode.WithModel("claude-sonnet-4")
```

**OpenCode:**
```bash
opencode run '...' --model openrouter/anthropic/claude-sonnet-4
opencode run '...' --model opencode/big-pickle    # OpenCode 内置模型
opencode run '...' --variant high                  # 推理变体
```

### 4.4 工作目录对比

**Claude Code:**
```go
claudecode.WithCwd("/path/to/dir")
```

**OpenCode:**
```bash
opencode run '...' --dir /path/to/dir
```

### 4.5 环境变量认证对比

**Claude Code:**
```go
claudecode.WithEnv(map[string]string{
    "ANTHROPIC_API_KEY": "sk-xxx",
    "ANTHROPIC_BASE_URL": "https://api.anthropic.com",
})
```

**OpenCode:**
```bash
ANTHROPIC_AUTH_TOKEN=sk-xxx \
ANTHROPIC_BASE_URL=https://api.anthropic.com \
opencode run '...'
```

---

## 五、OpenCode CLI 路由方案

OpenCode 支持两种运行模式：

### 5.1 单次任务模式 (`opencode run`)

适合一次性任务，执行完自动退出：

```bash
opencode run "Create a REST API endpoint" --format json --dir /path/to/project
```

特点：
- 无需 PTY
- JSON 流式输出
- 执行完成自动退出
- 适合单轮对话场景

### 5.2 交互式模式 (`opencode`)

适合需要多轮交互的任务：

```bash
opencode --dir /path/to/project  # 启动 TUI
```

特点：
- 需要 PTY（伪终端）
- 通过 stdin/stdout 交互
- 支持 Ctrl+C 退出
- 适合复杂多轮任务

---

## 六、架构设计

### 6.1 目录结构

参考 `infrastructure/claudecode/`，新建：

```
backend/infrastructure/opencode/
├── types.go              # OpenCodeProcessor、Interface、Session
├── processor_main.go     # Process / ProcessWithStreaming 主逻辑
├── query_streaming.go    # 解析 JSON Lines 流
├── hooks.go              # 工具钩子适配器
├── options.go            # CLI 参数构建
└── provider.go           # Provider 解析
```

### 6.2 核心接口设计

```go
// StreamingCallback 与 ClaudeCodeProcessor 相同
type StreamingCallback interface {
    OnThinking(thinking string)
    OnToolCall(toolName string, input map[string]any)
    OnToolResult(toolName string, result string)
    OnText(text string)
    OnComplete(finalResult string)
    GetFinalResult() string
}

// OpenCodeSession 会话上下文
type OpenCodeSession struct {
    SessionKey   string
    CliSessionID string
    WorkDir      string
}

// OpenCodeProcessorInterface
type OpenCodeProcessorInterface interface {
    Process(ctx context.Context, msg *bus.InboundMessage, session *OpenCodeSession, agent *domain.Agent) (string, error)
    ProcessWithStreaming(ctx context.Context, msg *bus.InboundMessage, session *OpenCodeSession, agent *domain.Agent, callback StreamingCallback) error
}
```

### 6.3 CLI 参数映射

```go
func buildOptions(provider *domain.LLMProvider, agent *domain.Agent) []string {
    args := []string{"run"}

    // 模型
    if agent.OpenCodeConfig().Model != "" {
        args = append(args, "--model", agent.OpenCodeConfig().Model)
    } else if provider != nil {
        // 从 provider 构建模型名
        args = append(args, "--model", provider.Model())
    }

    // 工作目录
    if agent.OpenCodeConfig().WorkDir != "" {
        args = append(args, "--dir", agent.OpenCodeConfig().WorkDir)
    }

    // 会话
    if sessionID != "" {
        args = append(args, "--session", sessionID)
    }

    // 继续会话
    if agent.OpenCodeConfig().Continue {
        args = append(args, "--continue")
    }

    // 分叉会话
    if agent.OpenCodeConfig().Fork {
        args = append(args, "--fork")
    }

    // Agent 类型
    if agent.OpenCodeConfig().AgentType != "" {
        args = append(args, "--agent", agent.OpenCodeConfig().AgentType)
    }

    // 权限跳过
    if agent.OpenCodeConfig().SkipPermissions {
        args = append(args, "--dangerously-skip-permissions")
    }

    // 思考过程
    if agent.OpenCodeConfig().ShowThinking {
        args = append(args, "--thinking")
    }

    // 分享会话
    if agent.OpenCodeConfig().ShareSession {
        args = append(args, "--share")
    }

    // 模型变体
    if agent.OpenCodeConfig().Variant != "" {
        args = append(args, "--variant", agent.OpenCodeConfig().Variant)
    }

    // 输出格式（固定为 JSON）
    args = append(args, "--format", "json")

    args = append(args, "--", userMessage)
    return args
}
```

### 6.4 JSON 流解析

```go
func (p *OpenCodeProcessor) parseJSONStream(ctx context.Context, reader *bufio.Reader, callback StreamingCallback) error {
    scanner := bufio.NewScanner(bufio.MaxScanTokenSize * 64) // 增大扫描缓冲区
    for scanner.Scan() {
        line := scanner.Bytes()
        if len(line) == 0 {
            continue
        }

        var event OpenCodeEvent
        if err := json.Unmarshal(line, &event); err != nil {
            p.logger.Warn("Failed to parse JSON line", zap.String("line", string(line)), zap.Error(err))
            continue
        }

        switch event.Type {
        case "text":
            callback.OnText(event.Part.Text)
        case "thinking":
            callback.OnThinking(event.Part.Thinking)
        case "tool_use":
            // 工具调用先通知（input），再通知结果（output/error）
            callback.OnToolCall(event.Part.Tool, event.Part.State.Input)
            if event.Part.State.Status == "completed" {
                callback.OnToolResult(event.Part.Tool, event.Part.State.Output)
            } else if event.Part.State.Status == "error" {
                errMsg := ""
                if event.Part.State.Error != nil {
                    errMsg = fmt.Sprintf("Error: %v", *event.Part.State.Error)
                }
                callback.OnToolResult(event.Part.Tool, errMsg)
            }
        case "step_finish":
            if event.Part.Reason == "stop" {
                callback.OnComplete(callback.GetFinalResult())
                return nil
            }
        }
    }
    return scanner.Err()
}
```

### 6.5 Domain 层变更

```go
// domain/agent.go
const (
    AgentTypeBareLLM  AgentType = "BareLLM"
    AgentTypeCoding   AgentType = "CodingAgent"
    AgentTypeOpenCode AgentType = "OpenCodeAgent"  // 新增
)

// domain/opencode_config.go
type OpenCodeConfig struct {
    Model           string            // 模型，如 openrouter/anthropic/claude-sonnet-4
    WorkDir         string            // 工作目录
    SystemPrompt    string            // 系统提示词（通过环境变量传递）
    AgentType       string            // build | plan | explore | general 等
    Continue        bool              // 是否继续上次会话
    Fork            bool              // 是否分叉会话
    SessionID       string            // 指定会话 ID
    SkipPermissions bool              // 跳过权限确认
    ShowThinking    bool              // 显示思考过程
    ShareSession    bool              // 分享会话
    Variant         string            // 模型变体：high | max | minimal
    Env             map[string]string // 环境变量
}
```

### 6.6 MessageProcessor 变更

```go
// pkg/channel/message_processor.go
type MessageProcessor struct {
    // ... 现有字段
    openCodeProcessor opencode.OpenCodeProcessorInterface  // 新增
}

// pkg/channel/response_generation.go
if agent != nil && agent.AgentType().String() == "OpenCodeAgent" {
    err := p.openCodeProcessor.ProcessWithStreaming(ctx, msg, openSession, agent, callback)
} else if agent != nil && agent.AgentType().String() == "CodingAgent" {
    err := p.claudeCodeProcessor.ProcessWithStreaming(ctx, msg, ccSession, agent, callback)
}
```

---

## 七、待验证项

以下功能需要进一步测试验证：

### 7.1 系统提示词传递

OpenCode CLI 不提供 `--system-prompt` 参数。可能的替代方案：
- 环境变量 `OPENCODE_SYSTEM_PROMPT`（需验证）
- 通过 `--file` 附加包含系统提示的文件
- 修改 OpenCode 配置文件

### 7.2 工具过滤

OpenCode CLI 不提供 `--allowed-tools` 或 `--disallowed-tools` 参数。解决方案：
- 选择受限的 Agent 类型（如 `plan`、`explore`）
- 使用 `--dangerously-skip-permissions` 跳过确认
- 通过 MCP 服务器封装工具集

### 7.3 超时控制

OpenCode CLI 不提供内置超时参数。解决方案：
- 通过 Go context 的 `exec.CommandContext` 实现外部超时
- 在 wrapper script 中使用 `timeout` 命令

### 7.4 MCP 服务器

OpenCode 提供 `opencode mcp` 命令管理 MCP 服务器：
- 配置存储在 OpenCode 配置目录
- 运行时自动加载 MCP 服务器

### 7.5 Session 持久化

OpenCode session 数据存储在 `~/.local/share/opencode`，可使用 `opencode session list` 查看。

---

## 八、限制与风险

| 限制项 | 影响 | 缓解方案 |
|--------|------|----------|
| 无 SDK，CLI 封装 | 稳定性依赖 CLI 输出格式 | 定期测试 CLI 输出格式兼容性 |
| 系统提示词不直接支持 | 无法直接自定义 Agent 行为 | 通过环境变量或配置文件 |
| 工具过滤不直接支持 | 无法细粒度控制工具 | 选择受限 Agent 类型 |
| 无内置超时 | 可能长时间阻塞 | 使用 `exec.CommandContext` |
| 工具调用协议差异 | Hook 适配器需要重写 | 参考 Claude Code 实现 |

---

## 九、结论

### 9.1 可行性评估

OpenCode CLI 集成在技术层面**完全可行**，核心功能（消息处理、工具调用、流式输出）均可实现。

### 9.2 功能覆盖率

预计可实现 Claude Code 约 **90%** 的核心功能：

| ✅ 可实现 | ⚠️ 受限支持 |
|---------|------------|
| 消息处理 | 系统提示词（环境变量） |
| 流式文本输出 | 工具过滤（Agent 类型） |
| 思考过程展示 | 超时控制（外部） |
| 工具调用拦截 | |
| Session 管理 | |
| 模型选择 | |
| 工作目录设置 | |
| 权限跳过 | |
| Agent 类型 | |
| 会话分叉 | |
| 用量统计 | |
| MCP 服务器 | |

### 9.3 建议

1. **MVP 版本**：先实现 `opencode run` 单次任务模式，覆盖核心对话和工具调用功能
2. **后续迭代**：根据实际使用情况，补齐 Session 管理、超时控制等高级功能
3. **关注 CLI 更新**：OpenCode CLI 仍在活跃开发，新版本可能增加更多功能

---

## 十、附录

### 10.1 OpenCode 路径

```
Home:     ~/.local/share/opencode
Data:     ~/.local/share/opencode
Bin:      ~/.cache/opencode/bin
Log:      ~/.local/share/opencode/log
Cache:    ~/.cache/opencode
Config:   ~/.config/opencode
State:    ~/.local/state/opencode
```

### 10.2 测试命令

```bash
# 基本测试
opencode run "Respond with: OK" --format json

# 带环境变量认证
ANTHROPIC_AUTH_TOKEN=$ANTHROPIC_AUTH_TOKEN \
ANTHROPIC_BASE_URL=$ANTHROPIC_BASE_URL \
opencode run "Respond with: OK" --format json

# 带工具调用
opencode run "List files in /tmp" --format json --dir /tmp --dangerously-skip-permissions

# 会话管理
opencode session list
opencode run "Remember this" --format json  # 创建会话
opencode run "Continue" -c --format json     # 继续会话
```

---

## 参考资料

- [OpenCode GitHub](https://github.com/anomaly Theorem/opencode)
- [OpenCode 官方文档](https://opencode.ai)
- `hermes-agent/skills/autonomous-ai-agents/opencode/SKILL.md`