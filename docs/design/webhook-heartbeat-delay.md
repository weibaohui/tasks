# Webhook 心跳延迟执行功能设计

## 1. 背景

当事件触发类心跳同时触发多条时，需要错开执行时间，避免相互等待。增加延迟选项，实现错峰执行。

## 2. 方案

### 2.1 领域模型变更

**WebhookHeartbeatBinding** 增加字段：
```go
type WebhookHeartbeatBinding struct {
    // ... 现有字段
    delayMinutes int  // 延迟执行分钟数，0表示不延迟
}
```

### 2.2 数据库变更

**表 `webhook_heartbeat_bindings`** 增加列：
```sql
ALTER TABLE webhook_heartbeat_bindings ADD COLUMN delay_minutes INTEGER NOT NULL DEFAULT 0;
```

### 2.3 服务层变更

**GitHubWebhookService.HandleWebhookEvent()**：
- 查询 binding 时获取 `delay_minutes`
- 若 `delay_minutes > 0`，使用 goroutine + timer 延迟执行
- 若 `delay_minutes == 0`，立即执行（现有逻辑）

```go
// 伪代码
if binding.delayMinutes > 0 {
    time.AfterFunc(time.Duration(binding.delayMinutes)*time.Minute, func() {
        triggerService.TriggerWithSource(...)
    })
} else {
    triggerService.TriggerWithSource(...)
}
```

### 2.4 前端变更

**WebhookHeartbeatBinding 编辑界面**：
- 新增延迟选项选择器（预设选项：3、5、10、15、20 分钟）
- 默认值：0（不延迟）

**预设选项**：`[0, 3, 5, 10, 15, 20]`，0 表示不延迟

## 3. 实现步骤

| 步骤 | 内容 | 文件 |
|------|------|------|
| 1 | 领域模型增加 delayMinutes 字段 | `backend/domain/github_webhook.go` |
| 2 | 数据库表增加 delay_minutes 列 | `backend/infrastructure/persistence/schema_tables.go` |
| 3 | Repository 增加字段读写 | `backend/infrastructure/persistence/webhook_heartbeat_binding_repository.go` |
| 4 | Service 层增加延迟执行逻辑 | `backend/application/github_webhook_service.go` |
| 5 | 前端类型增加字段 | `frontend/src/types/githubWebhook.ts` |
| 6 | 前端 API 增加字段 | `frontend/src/api/githubWebhookApi.ts` |
| 7 | 前端 UI 增加延迟选项 | `frontend/src/pages/ProjectWebhookPage.tsx` |

## 4. 测试要点

- 延迟字段正确保存和读取
- 延迟执行确实生效（sleep 后才执行）
- 多个心跳同时触发时能正确错峰
- 不设置延迟时行为与原来一致
