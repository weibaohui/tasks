# 多方式心跳触发设计文档

## 总体架构

核心设计是将心跳的"调度"与"执行"职责分离：
- `HeartbeatScheduler` 只负责按 cron 规则"调度"；
- 新建的 `HeartbeatTriggerService` 负责心跳的"执行"（创建需求、初始化状态机、派发需求）。

这样，HTTP Webhook、CLI 命令、状态机 Hook 都可以直接调用 `HeartbeatTriggerService.Trigger()`，而无需关心 cron 调度。

## 关键变更

### 1. HeartbeatTriggerService 提取

新建 `backend/application/heartbeat_trigger_service.go`：
- 持有心跳执行所需的全部依赖（repository、dispatch service、state machine service 等）；
- `Trigger(ctx, heartbeatID)` 方法封装完整的执行流程，逻辑从 `HeartbeatScheduler.executeHeartbeat` 迁移而来。

### 2. HTTP Webhook

在 `backend/interfaces/http/heartbeat_handler.go` 中新增 `TriggerHeartbeat` handler：
- 路由：`POST /api/v1/heartbeats/:id/trigger`
- 读取路径参数 `id`，调用 `HeartbeatTriggerService.Trigger()`
- 成功返回 `{"message": "triggered"}`

### 3. CLI 命令

- `backend/cmd/cli/client/client_heartbeat.go`：新增 `TriggerHeartbeat(ctx, heartbeatID)` 方法，发送 POST 请求；
- `backend/cmd/cli/cmd/project_heartbeat.go`：新增 `trigger` 子命令，调用客户端方法。

### 4. 状态机 Hook 扩展

- `backend/domain/statemachine/state_machine.go`：`TransitionHook.Type` 支持 `trigger_heartbeat`；
- `backend/infrastructure/statemachine/transition_executor.go`：
  - 新增 `heartbeatTrigger` 接口依赖；
  - `executeHook` switch 中增加 `case "trigger_heartbeat"`；
  - 从 `hook.Config["heartbeat_id"]` 读取 ID 并调用触发服务。

### 5. main.go 依赖注入

- 先创建 `HeartbeatTriggerService`；
- 将其注入到 `HeartbeatScheduler`（替换内部自行构造）和 `HeartbeatHandler`；
- 同时注入到 `TransitionExecutor`，使状态机 Hook 能够触发心跳。

## 时序图

```
外部请求/调度器/状态机
         |
         v
+-------------------------+
| HeartbeatTriggerService |
|        .Trigger()       |
+------------+------------+
             |
    +--------+--------+--------+
    |                 |        |
    v                 v        v
heartbeatRepo    projectRepo  requirementRepo
    |                 |        |
    v                 v        v
NewRequirement  RenderPrompt   Save
    |                          |
    +----------+---------------+
               |
               v
    +---------------------+
    | StateMachineService | (optional)
    | InitializeRequirementState
    +---------------------+
               |
               v
    +-------------------------+
    | RequirementDispatchService
    |    .DispatchRequirement()
    +-------------------------+
```
