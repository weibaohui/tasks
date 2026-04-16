# 多方式心跳触发需求文档

## 背景

当前心跳（Heartbeat）仅支持基于 cron 的时间触发（`interval_minutes`）。在实际使用中，用户希望能够通过更多方式触发心跳，以适应不同的集成场景。

## 目标

扩展心跳触发机制，支持以下三种额外的触发方式：
1. **Webhook 触发**：外部系统通过 HTTP POST 请求触发指定心跳；
2. **CLI 子命令触发**：通过命令行手动触发一次心跳；
3. **状态机 Hook 触发**：状态机转换时，支持将触发心跳作为 transition hook 的一种类型。

## 需求详情

### 1. Webhook 触发

- 提供 HTTP API 端点 `POST /api/v1/heartbeats/:id/trigger`；
- 调用该端点后，应立即执行对应心跳的完整流程（创建需求、初始化状态机、派发需求）；
- 复用现有的 Bearer Token 认证机制。

### 2. CLI 子命令触发

- 提供 CLI 命令 `taskmanager project heartbeat trigger <heartbeat_id>`；
- 该命令调用上述 Webhook API，手动触发一次心跳执行。

### 3. 状态机 Hook 触发

- 扩展状态机的 `TransitionHook` 类型，新增 `trigger_heartbeat`；
- 在状态机配置中，允许为某个 transition 配置 `trigger_heartbeat` hook，指定要触发的心跳 ID；
- 当状态转换发生时，自动触发对应的心跳。

## 非功能需求

- 三种触发方式应复用同一套心跳执行逻辑，避免代码重复；
- 不修改 `heartbeats` 表 Schema；
- 核心逻辑必须有单元测试覆盖；
- `go build ./...` 编译通过，测试通过。
