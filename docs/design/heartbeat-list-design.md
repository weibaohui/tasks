# 心跳列表改造设计文档

## 概述

本文档描述如何将项目级别的单心跳配置改造为独立的心跳列表（Heartbeat List），实现每个心跳专注单一任务、独立调度。

## 架构约束

- 后端遵循 DDD 分层：domain → application → infrastructure → interfaces
- 前端使用 React + TypeScript + Ant Design
- `Heartbeat` 作为独立聚合，拥有独立的生命周期和仓储
- `projects` 表旧列保留（兼容旧数据/外部脚本），但代码层不再读写

## 设计方案

### 1. 后端领域层（Domain）

#### 1.1 新增 `domain/heartbeat.go`

```go
type HeartbeatID struct { value string }
func NewHeartbeatID(value string) HeartbeatID { ... }

type Heartbeat struct {
    id              HeartbeatID
    projectID       ProjectID
    name            string
    enabled         bool
    intervalMinutes int
    mdContent       string
    agentCode       string
    requirementType string
    sortOrder       int
    createdAt       time.Time
    updatedAt       time.Time
}
```

行为方法：
- `NewHeartbeat(id HeartbeatID, projectID ProjectID, name string, intervalMinutes int, mdContent, agentCode, requirementType string) (*Heartbeat, error)`
  - 校验 `name` 非空、`intervalMinutes >= 1`、`agentCode` 非空
- `Update(name string, intervalMinutes int, mdContent, agentCode, requirementType string) error`
  - 同样执行上述校验
- `SetEnabled(bool)` / `SetSortOrder(int)`
- `RenderPrompt(project *Project) string`：替换 `${project.id}`、`${project.name}` 等变量
- Getters：`ID()`、`ProjectID()`、`Name()`、`Enabled()`、`IntervalMinutes()`、`MDContent()`、`AgentCode()`、`RequirementType()`、`SortOrder()`、`CreatedAt()`、`UpdatedAt()`
- `ToSnapshot()` / `FromSnapshot()`

#### 1.2 新增 `domain/heartbeat_repository.go`

```go
type HeartbeatRepository interface {
    Save(ctx context.Context, hb *Heartbeat) error
    FindByID(ctx context.Context, id HeartbeatID) (*Heartbeat, error)
    FindByProjectID(ctx context.Context, projectID ProjectID) ([]*Heartbeat, error)
    FindAllEnabled(ctx context.Context) ([]*Heartbeat, error)
    Delete(ctx context.Context, id HeartbeatID) error
}
```

#### 1.3 改造 `domain/project.go`

移除字段：
- `heartbeatEnabled`
- `heartbeatIntervalMinutes`
- `heartbeatMDContent`
- `agentCode`

移除方法：
- `HeartbeatEnabled()`
- `HeartbeatIntervalMinutes()`
- `HeartbeatMDContent()`
- `AgentCode()`
- `UpdateHeartbeatConfig(...)`

`ProjectSnapshot` 同步移除上述字段。

### 2. 后端基础设施层（Infrastructure）

#### 2.1 新增表 `backend/infrastructure/persistence/schema_tables.go`

```sql
CREATE TABLE IF NOT EXISTS heartbeats (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    name TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    interval_minutes INTEGER NOT NULL DEFAULT 60,
    md_content TEXT NOT NULL DEFAULT '',
    agent_code TEXT NOT NULL DEFAULT '',
    requirement_type TEXT NOT NULL DEFAULT 'heartbeat',
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_heartbeats_project_id ON heartbeats(project_id);
CREATE INDEX IF NOT EXISTS idx_heartbeats_enabled ON heartbeats(enabled);
```

#### 2.2 新增 `backend/infrastructure/persistence/heartbeat_repository.go`

实现 `HeartbeatRepository`：
- `Save`：INSERT OR REPLACE
- `FindByID` / `FindByProjectID` / `FindAllEnabled`
- `Delete`

#### 2.3 新增迁移 `backend/infrastructure/persistence/migrations.go`

```go
func MigrateHeartbeatToTable(db *sql.DB) error
```

逻辑：
1. 检查并创建 `heartbeats` 表
2. 查询所有 `heartbeat_enabled = 1 AND agent_code != ''` 的项目
3. 为每个这样的项目插入一条默认心跳：
   - `id = "hb_" + project.id`
   - `name = "默认心跳"`
   - `interval_minutes = project.heartbeat_interval_minutes`
   - `md_content = project.heartbeat_md_content`
   - `agent_code = project.agent_code`
   - `requirement_type = "heartbeat"`
   - `sort_order = 0`

#### 2.4 改造 `backend/infrastructure/persistence/project_repository.go`

- `Save`：移除 `heartbeat_enabled`、`heartbeat_interval_minutes`、`heartbeat_md_content`、`agent_code` 的 INSERT/UPDATE
- `FindByID` / `FindAll`：SELECT 移除上述列
- `scanProject`：移除扫描和快照赋值

### 3. 后端应用层（Application）

#### 3.1 新增 `backend/application/heartbeat_service.go`

`HeartbeatApplicationService` 提供 CRUD：

```go
type CreateHeartbeatCommand struct {
    ProjectID       string
    Name            string
    IntervalMinutes int
    MDContent       string
    AgentCode       string
    RequirementType string
}

type UpdateHeartbeatCommand struct {
    ID              string
    Name            string
    IntervalMinutes int
    MDContent       string
    AgentCode       string
    RequirementType string
    Enabled         bool
}
```

方法：
- `CreateHeartbeat(ctx, cmd) (*Heartbeat, error)`
- `UpdateHeartbeat(ctx, cmd) (*Heartbeat, error)`
- `DeleteHeartbeat(ctx, id string) error`
- `ListHeartbeatsByProject(ctx, projectID string) ([]*Heartbeat, error)`
- `GetHeartbeat(ctx, id string) (*Heartbeat, error)`

每次 Create/Update/Delete 后调用 `scheduler.RefreshSchedule(ctx, heartbeatID)`。

#### 3.2 改造 `backend/application/heartbeat_scheduler.go`

调度器改为按心跳调度：

```go
type HeartbeatScheduler struct {
    cron                       *cron.Cron
    heartbeatRepo              domain.HeartbeatRepository
    projectRepo                domain.ProjectRepository
    requirementRepo            domain.RequirementRepository
    idGenerator                domain.IDGenerator
    inboundPublisher           interface{ PublishInbound(msg *channelBus.InboundMessage) }
    requirementDispatchService *RequirementDispatchService
    stateMachineService        *StateMachineService
    rootCtx                    context.Context
    // cron 任务记录：heartbeatID -> cron.EntryID
    entries                    map[string]cron.EntryID
}
```

- `Start(ctx)`：调用 `heartbeatRepo.FindAllEnabled(ctx)`，为每个心跳调用 `scheduleHeartbeat(hb)`
- `scheduleHeartbeat(hb *Heartbeat)`：生成 cron 表达式，注册到 `s.cron`，记录 `entries[hb.ID().String()]`
- `RefreshSchedule(ctx, heartbeatID string)`：
  - 若 entry 存在则 `Remove(entryID)`
  - 重新加载该心跳
  - 若 `enabled` 则重新 `scheduleHeartbeat`
- `executeHeartbeat(ctx, heartbeatID string)`：
  1. `heartbeatRepo.FindByID`
  2. `projectRepo.FindByID`
  3. `hb.RenderPrompt(project)`
  4. `domain.NewRequirement(...)`，设置 `requirement_type = hb.RequirementType()`
  5. 保存需求、初始化状态机、调用 `DispatchRequirement`

移除方法：`UpdateProjectHeartbeat`、`scheduleProject`（按项目调度的旧逻辑）。

### 4. 后端接口层（Interfaces）

#### 4.1 新增 `backend/interfaces/http/heartbeat_handler.go`

```go
type HeartbeatHandler struct {
    service    *application.HeartbeatApplicationService
    scheduler  *application.HeartbeatScheduler
}
```

API：
- `GET /api/heartbeats?project_id=xxx` → `ListHeartbeats`
- `POST /api/heartbeats` → `CreateHeartbeat`
- `GET /api/heartbeats/:id` → `GetHeartbeat`
- `PUT /api/heartbeats/:id` → `UpdateHeartbeat`
- `DELETE /api/heartbeats/:id` → `DeleteHeartbeat`

响应体统一使用 `map[string]interface{}`。

#### 4.2 改造 `backend/interfaces/http/project_handler.go`

- `UpdateProjectRequest` 移除：`HeartbeatEnabled`、`HeartbeatIntervalMinutes`、`HeartbeatMDContent`、`AgentCode`
- `UpdateProject` handler 中不再传这些字段
- `projectToMap` 移除心跳相关字段
- `UpdateProject` 中不再调用 `heartbeatScheduler.RefreshSchedule`

#### 4.3 路由注册 `backend/interfaces/http/router.go`

新增 Heartbeat 路由组。

#### 4.4 CLI 改造 `backend/cmd/cli/cmd/project_heartbeat.go`

保留 `project heartbeat` 入口，子命令替换为基于心跳 ID 的操作：
- `list <project_id>`
- `create <project_id> --name <name> --interval <m> --agent-code <code> --type <type> [--content <md>]`
- `update <heartbeat_id> [--name <name>] [--interval <m>] [--agent-code <code>] [--type <type>] [--content <md>] [--enabled true/false]`
- `delete <heartbeat_id>`
- `enable <heartbeat_id>` / `disable <heartbeat_id>`

客户端 `backend/cmd/cli/client/client_project.go` 新增对应 HTTP 客户端方法。

### 5. 前端实现

#### 5.1 新增类型 `frontend/src/types/heartbeat.ts`

```ts
export interface Heartbeat {
  id: string;
  project_id: string;
  name: string;
  enabled: boolean;
  interval_minutes: number;
  md_content: string;
  agent_code: string;
  requirement_type: string;
  sort_order: number;
  created_at: number;
  updated_at: number;
}
```

#### 5.2 新增 API `frontend/src/api/heartbeat.ts`

封装增删改查请求。

#### 5.3 改造 `frontend/src/pages/ProjectRequirementPage.tsx`

在项目配置的 Drawer/Modal 中：
- 移除原有的 `HeartbeatTemplateEditor` 直接嵌入
- 新增"心跳管理" Tab（或折叠面板）
- 使用 Table 展示心跳列表，列：名称、间隔、Agent、类型、启用状态、操作
- 操作按钮：编辑、删除、开关
- 顶部"新增心跳"按钮

#### 5.4 新增组件 `frontend/src/components/HeartbeatList/index.tsx`

心跳列表管理组件：
- Props：`projectId`、`heartbeats`、`onChange`
- 内部维护排序状态

#### 5.5 新增弹窗 `frontend/src/components/HeartbeatEditorModal/index.tsx`

编辑心跳的弹窗：
- Form 字段：名称、间隔（InputNumber min=1）、Agent（Select 从 agents 列表选）、需求类型（Select）、Prompt（TextArea）、启用开关
- 根据需求类型切换默认模板（从常量中获取）

#### 5.6 改造默认模板

将 `HeartbeatTemplate/index.tsx` 中的单一默认模板拆分为按类型的模板字典，供编辑器使用。原组件可改造为通用的模板编辑器，或复用 TextArea。

### 6. `backend/cmd/server/main.go` 与依赖注入

- `main.go` 中创建 `SQLiteHeartbeatRepository`
- `HeartbeatScheduler` 注入 `heartbeatRepo`
- `HeartbeatApplicationService` 注入 `heartbeatRepo`、`idGenerator`、`scheduler`
- 注册 `HeartbeatHandler`
- 启动时调用 `MigrateHeartbeatToTable(db)`

### 7. 数据流

```
旧项目数据
    ↓
MigrateHeartbeatToTable ──→ heartbeats 表（默认心跳记录）
    ↓
HeartbeatScheduler.Start() ──→ 为每个 enabled 心跳注册 cron 任务
    ↓
cron 触发 ──→ executeHeartbeat(heartbeatID)
    ↓
加载 project + 渲染 prompt ──→ 创建 Requirement（指定 requirement_type）
    ↓
保存 → 初始化状态机 → DispatchRequirement
```

### 8. 向后兼容说明

- `projects` 表旧列保留，旧版 CLI/外部脚本读取不会报错
- 代码层不再读取旧列，调度完全基于 `heartbeats` 表
- 迁移函数幂等：对已迁移的项目不重复插入（通过 `id = "hb_"+project_id` 唯一键保证，或先查后插）
