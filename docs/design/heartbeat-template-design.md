# 心跳模板管理设计文档

## 概述

在心跳管理弹窗中集成模板选择和应用能力，使用户能够快速创建相似的心跳任务。

## 数据模型

### 后端 Domain

```go
// HeartbeatTemplateID 值对象
type HeartbeatTemplateID struct{ value string }

// HeartbeatTemplate 实体
type HeartbeatTemplate struct {
    id              HeartbeatTemplateID
    name            string
    mdContent       string
    requirementType string
    createdAt       time.Time
    updatedAt       time.Time
}
```

### 数据库表

```sql
CREATE TABLE IF NOT EXISTS heartbeat_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    md_content TEXT NOT NULL DEFAULT '',
    requirement_type TEXT NOT NULL DEFAULT 'heartbeat',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
```

## 架构设计

### 后端分层

| 层级 | 文件 | 职责 |
|------|------|------|
| Domain | `domain/heartbeat_template.go` | 实体、值对象、构造器、验证、快照 |
| Domain | `domain/heartbeat_template_repository.go` | 仓储接口 |
| Infrastructure | `infrastructure/persistence/heartbeat_template_repository.go` | SQLite 实现 |
| Application | `application/heartbeat_template_service.go` | CRUD 应用服务 |
| Interfaces | `interfaces/http/heartbeat_template_handler.go` | REST API Handler |

### API 设计

- `GET /api/v1/heartbeat-templates` — 列出所有模板
- `POST /api/v1/heartbeat-templates` — 创建模板
  - Body: `{ "name": "string", "md_content": "string", "requirement_type": "string" }`
- `DELETE /api/v1/heartbeat-templates/:id` — 删除模板

### 前端交互设计

在 `HeartbeatManagement` 的 Modal 中，模板编辑器 (`HeartbeatTemplateEditor`) 上方增加模板操作行：

```
[选择模板... ▼]  [保存为模板]
```

- **选择模板**：`Select` 组件，加载全局模板列表。选择后自动填充 `md_content` 和 `requirement_type`。
- **保存为模板**：`Button` 组件，点击后弹出小型 `Modal`，输入模板名称，提交后调用创建 API，成功后刷新模板列表并清空选择框。

## Pet Shop 心跳拆分方案

将原来包含 3 个任务的默认心跳拆分为 3 条独立心跳：

| 心跳名称 | requirement_type | Prompt 来源 |
|----------|------------------|-------------|
| 派发需求 | normal | `DEFAULT_HEARTBEAT_TEMPLATE` 中"任务一"部分 |
| 处理 PR | pr_review | `DEFAULT_HEARTBEAT_TEMPLATE` 中"任务二"部分 |
| 提出优化点 | optimization | `DEFAULT_HEARTBEAT_TEMPLATE` 中"任务三"部分 |

同时预置这 3 个内容为全局心跳模板，方便其他项目复用。
