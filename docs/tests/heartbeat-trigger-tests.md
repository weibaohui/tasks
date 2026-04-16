# 多方式心跳触发测试文档

## 测试策略

### 1. HeartbeatTriggerService 单元测试

文件：`backend/application/heartbeat_trigger_service_test.go`

覆盖场景：
- `Trigger` 成功执行完整流程（加载心跳、加载项目、创建需求、保存需求、派发需求）；
- `Trigger` 当心跳不存在时返回错误；
- `Trigger` 当心跳未启用（`enabled=false`）时返回错误；
- `Trigger` 当项目不存在时返回错误。

### 2. TransitionExecutor 单元测试

文件：`backend/infrastructure/statemachine/transition_executor_test.go`

覆盖场景：
- 执行 `trigger_heartbeat` 类型 hook 成功；
- `trigger_heartbeat` hook 配置缺少 `heartbeat_id` 时失败；
- `heartbeatTrigger` 接口未注入时失败；
- `trigger_heartbeat` 触发失败时按 retry 配置重试。

### 3. HeartbeatHandler 路由测试

文件：`backend/interfaces/http/heartbeat_handler_test.go`（如已存在则追加，否则新建）

覆盖场景：
- `POST /api/v1/heartbeats/:id/trigger` 成功返回 200；
- 触发失败时返回对应错误状态码（如 400/404/500）。

### 4. 编译与集成验证

- `cd backend && go build ./...` 编译通过；
- `go test ./application/... ./infrastructure/statemachine/... -v` 测试通过；
- 启动服务后，通过 curl 和 CLI 手动验证触发行为。
