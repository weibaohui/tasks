# Hook 系统设计文档

## 概述

构建一个**可扩展、分层、事件驱动**的 Hook 系统，支持 LLM Agent 执行过程中所有关键节点的拦截、扩展和定制。

## 文档结构

| 文档 | 内容 |
|------|------|
| [01-requirements.md](./01-requirements.md) | 需求文档：功能需求、详细事件列表、验收标准 |
| [02-design.md](./02-design.md) | 设计文档：架构设计、核心接口、Registry/Executor 实现 |
| [03-implementation.md](./03-implementation.md) | 实现文档：代码实现、LLM 集成、使用示例 |
| [04-testing.md](./04-testing.md) | 测试文档：单元测试、集成测试、E2E 测试 |

## 快速开始

### 1. 创建 Manager

```go
logger, _ := zap.NewProduction()
manager := hook.NewManager(logger, nil)
```

### 2. 注册 Hooks

```go
manager.Register(hooks.NewLoggingHook(logger))
manager.Register(hooks.NewMetricsHook(logger))
manager.Register(hooks.NewRateLimitHook(rate.Limit(10), 20, logger))
```

### 3. 使用 Hooks

```go
// PreLLMCall
callCtx := &domain.LLMCallContext{Prompt: "Hello"}
modifiedCtx, err := manager.PreLLMCall(ctx, callCtx)

// PostLLMCall
resp := &domain.LLMResponse{Content: "response"}
modifiedResp, err := manager.PostLLMCall(ctx, callCtx, resp)
```

## Hook 分类

| 类别 | 事件数 | 描述 |
|------|--------|------|
| Lifecycle | 5 | 系统生命周期管理 |
| LLM | 8 | LLM 调用全流程拦截 |
| Tool | 10 | 工具执行全流程拦截 |
| Message | 5 | 消息处理拦截 |
| Skill | 6 | 技能系统拦截 |
| MCP | 6 | MCP 协议拦截 |
| Prompt | 6 | Prompt 管理拦截 |
| Session | 5 | 会话管理拦截 |
| **总计** | **51** | **8 大类** |

## 核心接口

```go
// Hook 基础接口
type Hook interface {
    Name() string
    Priority() int
    Enabled() bool
    SetEnabled(bool)
    HookType() HookType
}

// LLMHook 接口
type LLMHook interface {
    Hook
    PreLLMCall(ctx *HookContext, callCtx *LLMCallContext) (*LLMCallContext, error)
    PostLLMCall(ctx *HookContext, callCtx *LLMCallContext, response *LLMResponse) (*LLMResponse, error)
}

// ToolHook 接口
type ToolHook interface {
    Hook
    PreToolCall(ctx *HookContext, callCtx *ToolCallContext) (*ToolCallContext, error)
    PostToolCall(ctx *HookContext, callCtx *ToolCallContext, result *ToolExecutionResult) (*ToolExecutionResult, error)
    OnToolError(ctx *HookContext, callCtx *ToolCallContext, err error) (*ToolExecutionResult, error)
}
```

## 配置示例

```yaml
hook_system:
  enabled: true
  error_strategy: "continue"

  hooks:
    - name: logging
      enabled: true
      priority: 100

    - name: metrics
      enabled: true
      priority: 50

    - name: rate_limit
      enabled: true
      priority: 5
      config:
        max_calls_per_minute: 60
```

## 内置 Hooks

| Hook | 描述 |
|------|------|
| `LoggingHook` | 记录所有 Hook 调用日志 |
| `MetricsHook` | 收集 LLM 调用指标 |
| `RateLimitHook` | LLM 调用限流 |
| `CacheHook` | LLM 响应缓存（规划中） |

## 执行顺序

```
PreLLMCall Hooks (按优先级升序)
    ↓
LLM Actual Call
    ↓
PostLLMCall Hooks (按优先级降序)
```

## 错误处理策略

| 策略 | 描述 |
|------|------|
| `ErrorStrategyStopOnFirst` | 遇到错误立即停止 |
| `ErrorStrategyContinue` | 继续执行所有 Hook |

## 测试

```bash
# 运行所有测试
go test ./infrastructure/hook/... -v

# 运行性能测试
go test ./infrastructure/hook/... -bench=. -benchmem
```

## 项目状态

- [x] 需求分析
- [x] 架构设计
- [ ] 核心框架实现
- [ ] LLM Hooks 实现
- [ ] Tool Hooks 实现
- [ ] 单元测试
- [ ] 集成测试
- [ ] E2E 测试

## 术语表

| 术语 | 定义 |
|------|------|
| Hook | 拦截点在特定事件发生时被调用的回调函数 |
| Pre Hook | 在事件发生前执行的 Hook，可修改输入 |
| Post Hook | 在事件发生后执行的 Hook，可修改输出 |
| Hook Chain | 按优先级排序的 Hook 执行链 |
| Registry | Hook 注册表，管理所有 Hook 的注册和注销 |
| Executor | Hook 执行器，负责按顺序执行 Hook 链 |
