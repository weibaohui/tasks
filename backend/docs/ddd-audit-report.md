# DDD & Go 最佳实践审计报告

**审计日期**: 2026-04-12  
**审计范围**: `backend/` 全部 Go 源码（含全部子包）  
**审计维度**: DDD 分层规范 + Go 最佳实践

---

## 概览

| 级别 | 数量 | 说明 |
|------|------|------|
| P0 严重 | 4 | 破坏 DDD 分层原则或导致程序崩溃 |
| P1 重要 | 6 | 影响可维护性、可测试性或数据完整性 |
| P2 建议 | 5 | 可改进的代码组织、命名、注释问题 |

---

## P0 严重违规（必须修复）

### P0.1 `interfaces` 层直接引用 `infrastructure`

**问题**: interfaces 层直接持有基础设施的具体类型，违反了依赖倒置原则。

| 文件 | 违规导入 | 具体类型 |
|------|---------|----------|
| `interfaces/http/auth_handler.go:19` | `infrastructure/persistence` | `*persistence.SQLiteUserTokenRepository` |
| `interfaces/http/auth_handler.go:20` | `infrastructure/utils` | `*utils.NanoIDGenerator` |
| `interfaces/http/skill_handler.go:11` | `infrastructure/skill` | `*skill.SkillsLoader` |
| `interfaces/ws/handler.go:15` | `infrastructure/bus` | `*bus.EventBus` |

**修复方向**:
1. 在 `domain/` 或 `application/` 定义对应接口（如 `UserTokenRepository`、`IDGenerator`、`SkillsLoader`、`EventBus`）。
2. 让 interfaces 只依赖接口类型，从 `cmd/server/main.go` 注入具体实现。

---

### P0.2 `pkg/statemachine/sdk.go` 使用 `panic` 处理可恢复错误

**位置**: `backend/pkg/statemachine/sdk.go:82`, `:86`

```go
panic("failed to create db dir: " + err.Error())
panic("failed to open db: " + err.Error())
```

**问题**: `pkg/` 是被其他模块引用的公共库，使用 `panic` 会导致调用方程序直接崩溃，违反 Go 库代码最佳实践。

**修复方向**: 返回 `error` 让调用方决定如何处理；或改为懒加载并在 interface 层做容错提示。

---

### P0.3 `domain/factory.go` 的 `LLMProviderFactory.Build` 返回 `interface{}`

**位置**: `backend/domain/factory.go:8`

```go
Build(config *LLMProviderConfig) (interface{}, error)
```

**问题**: 领域层使用空接口抹杀类型安全，属于基础设施泄漏到领域的后门。

**修复方向**: 定义领域层明确的 `LLMProvider` 接口，基础设施实现该接口后返回具体类型。

---

### P0.4 `CreateAgent` 缺少 `IsActive` 字段导致功能退化

**位置**: `backend/application/agent_service.go:165`

**问题**: 在本次重构中，`CreateAgentCommand` 的 `IsActive` 字段被移除（或条件判断被误删），导致无法通过 API 显式创建非活跃 Agent。

**修复方向**: 恢复 `if cmd.IsActive != nil { agent.SetActive(*cmd.IsActive) }` 逻辑（注意 `CreateAgentCommand` 当前没有 `IsActive` 指针字段，需要补回）。

---

## P1 重要违规（建议尽快修复）

### P1.1 Persistence 层中 13 处 `_ = json.Unmarshal` 忽略反序列化错误

**问题**: 数据库字段反序列化失败被静默忽略，会导致数据 silently corrupted。

**涉及文件与行号**:
- `infrastructure/persistence/session_repository.go:149`
- `infrastructure/persistence/mcp_server_repository.go:146`, `:150`, `:154`
- `infrastructure/persistence/agent_mcp_binding_repository.go:115`
- `infrastructure/persistence/project_repository.go:116`
- `infrastructure/persistence/channel_repository.go:158`, `:160`
- `infrastructure/persistence/agent_repository.go:288`, `:289`, `:294`
- `infrastructure/persistence/llm_provider_repository.go:213`, `:215`

**修复方向**: 反序列化失败应返回 `fmt.Errorf("failed to unmarshal X: %w", err)`。

---

### P1.2 `requirement_dispatch_service.go` 清理 workspace 时忽略错误

**位置**: `backend/application/requirement_dispatch_service.go:120`, `:137`, `:145`, `:153`

```go
_ = s.workspaceManager.RemoveWorkspace(workspacePath)
```

**问题**: 分发失败的清理过程中错误被静默忽略，导致日志缺失、磁盘泄漏难以排查。

**修复方向**: 使用 `log.Printf` 记录清理错误，但不要因清理失败而阻塞主流程。

---

### P1.3 `mcp_server_repository.go` 等缺少 `EnvVars` 防御性拷贝

**问题**: `MCPServer.UpdateProfile` 已对 `EnvVars` 做拷贝，但仓库层 `FromSnapshot` 中还有路径可能直接赋值外部 map。

**修复方向**: 统一审查 persistence 中从数据库恢复对象时是否所有可变字段（`map`、`slice`）都进行了防御性拷贝。

---

### P1.4 超大文件未拆分（超过 500 行硬上限）

| 文件 | 行数 | 说明 |
|------|------|------|
| `cmd/cli/client/client.go` | 1814 | CLI HTTP 客户端，可拆为 `request.go`, `response.go`, `retry.go` |
| `infrastructure/hook/hooks/conversation_record.go` | 579 | Hook 实现，可按事件类型拆分 |
| `cmd/server/main.go` | 586 | DI 入口，可按功能模块拆分初始化 |
| `infrastructure/persistence/schema.go` | 462 | Schema 定义，可按实体拆分为多个 migration 文件 |
| `interfaces/http/requirement_handler.go` | 454 | Handler 方法可按子资源拆分 |
| `infrastructure/llm/eino.go` | 449 | LLM 实现，可按操作类型拆分 |

**修复方向**: 按功能职责拆分为多个同包文件，零行为变更。

---

### P1.5 `application` 层存在 `context.Background()` 硬编码

**位置**:
- `backend/application/trace_context.go:37`
- `backend/application/runtime.go:71`, `:73`
- `backend/application/heartbeat_scheduler.go:172`

**问题**: 后台任务和运行时创建根 context 时使用硬编码 `context.Background()`，导致无法通过外层取消信号统一控制生命周期。

**修复方向**: 通过构造函数注入 `context.Context` 或使用 `ctx context.Context` 作为方法参数传入。

---

### P1.6 `application/mcp_service.go` 使用 `fmt.Sprintf` 组装状态消息

**位置**: `backend/application/mcp_service.go:186`, `:196`, `:204`, `:225`, `:235`, `:243`, `:251`

```go
server.SetStatus("error", fmt.Sprintf("创建客户端失败: %v", err))
```

**问题**: application 层直接操作领域模型的内部状态（`SetStatus`），且错误消息格式属于表现层/技术细节。

**修复方向**: 定义 `MCPServer` 的 domain 方法（如 `server.MarkInitializationFailed(err error)`），让 domain 层自己决定如何记录错误摘要。

---

## P2 建议改进（可选）

### P2.1 `map[string]interface{}` 在 domain 层大量使用

**位置**: `domain/channel.go`, `domain/session.go`, `domain/statemachine/hook_executor.go`, `domain/statemachine/state_machine.go`

**问题**: `map[string]interface{}` 是弱类型，在 domain 层使用会降低类型安全性。

**修复方向**: 对于配置类字段（如 `Config`、`Metadata`），可保留当前设计（灵活度高）；对于结构明确的业务字段，建议逐步改为强类型结构体。

---

### P2.2 `cmd/cli/cmd/*` 中存在大量 `init()` 函数

**问题**: `init()` 函数虽然符合 Cobra 习惯，但数量过多（约 20 个），导致包导入时产生隐式副作用，单元测试难以控制。

**修复方向**: 对 CLI 命令使用显式注册表模式，在 `main.go` 中统一注册，减少 `init()` 依赖。

---

### P2.3 `TODO` 注释未清理

**位置**:
- `application/state_machine_service.go:230`
- `infrastructure/llm/claude.go:138`
- `infrastructure/llm/ollama.go:122`
- `infrastructure/statemachine/transition_executor.go:99`

**修复方向**: 标记的实现类 TODO 应尽快完成或转为正式的 Issue/需求文档。

---

### P2.4 `domain` 层缺少 `context.Context` 参数

**问题**: `domain` 层的方法（如 `Agent.UpdateProfile`、`Requirement.UpdateContent`）没有接收 `context.Context`。虽然 domain 层通常不直接发 HTTP 请求，但如果未来需要在 domain 中加入事件发布、校验调用外部服务等操作，会缺少上下文链路。

**修复方向**: 目前可保持现状；若后续引入领域事件总线，再统一为聚合根方法注入 `ctx`。

---

### P2.5 `requirement_dispatch_prompt.go` 496 行

**问题**: 虽然函数职责内聚（全部是 prompt 构建），但文件超过 500 行硬上限。

**修复方向**: 可按 "heartbeat prompt" / "coding prompt" / "state machine helpers" 拆分为 3 个文件。

---

## 正向实践（值得保持）

✅ **无循环依赖**: `go build ./...` 一遍通过。  
✅ **Application 层不依赖基础设施**: 普通 application service 没有直接 import `infrastructure/*`。  
✅ **Repository 接口定义在 Domain 层**: 所有仓储接口都在 `domain/repository.go`，方向正确。  
✅ **值对象不可变字段封装良好**: `AgentID.value` 等小写内部字段 + 导出 constructor/getter 的做法规范。  
✅ **防御性拷贝已成惯例**: `append([]string(nil), ...)` 在 getter 中被广泛使用。  
✅ **上下文参数位置规范**: 所有方法均遵循 `ctx context.Context` 作为第一个参数。

---

## 下一步行动计划（建议优先级）

1. **P0.1** > 新建 `domain` 接口，重构 4 个 interfaces handler 的依赖注入（1 天）。
2. **P0.2** > 将 `pkg/statemachine/sdk.go` 的 `panic` 改为 `error` 返回（0.5 天）。
3. **P1.1 + P1.2** > 批量修复 persistence 和 dispatch service 的忽略错误（0.5 天）。
4. **P1.4** > 按功能拆分 6 个超大文件（1-2 天）。
5. **P1.6 + P1.5** > 抽提 domain 方法和注入根 context（1 天）。
