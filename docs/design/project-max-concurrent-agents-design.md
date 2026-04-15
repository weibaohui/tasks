# 项目最大并发 Agent 数配置设计文档

## 概述

本文档描述如何在项目级别增加"最大并发 Agent 数"配置，并在需求派发时强制执行该限制。

## 架构约束

- 后端遵循 DDD 分层：domain → application → infrastructure → interfaces
- 前端使用 React + TypeScript + Ant Design
- 不破坏现有 API 契约，新增字段可空或带默认值

## 设计方案

### 1. 后端领域层（Domain）

**文件**：`backend/domain/project.go`

- `Project` struct 新增 `maxConcurrentAgents int` 字段
- `NewProject()` 中初始化默认值 `2`
- 新增 `MaxConcurrentAgents() int` getter
- 新增 `SetMaxConcurrentAgents(value int) error`：
  - `value < 1 || value > 10` → 返回 `ErrProjectMaxConcurrentAgentsInvalid`
  - 合法值 → 赋值并更新 `updatedAt`
- `ProjectSnapshot` 同步新增 `MaxConcurrentAgents int`
- `ToSnapshot()` / `FromSnapshot()` 包含新字段

### 2. 后端应用层（Application）

**文件**：`backend/application/project_service.go`

- `UpdateProjectCommand` 新增 `MaxConcurrentAgents *int`
- `UpdateProject()` 逻辑：
  ```go
  if cmd.MaxConcurrentAgents != nil {
      if err := project.SetMaxConcurrentAgents(*cmd.MaxConcurrentAgents); err != nil {
          return nil, err
      }
  }
  ```

**文件**：`backend/application/requirement_dispatch_service.go`

- 新增错误：`ErrMaxConcurrentAgentsReached = errors.New("max concurrent agents limit reached for project")`
- `DispatchRequirement()` 插入点：项目校验通过后、获取 `baseAgent` 之前
- 插入逻辑：
  ```go
  runningCount, err := s.requirementRepo.Count(ctx, domain.RequirementListFilter{
      ProjectID: &requirement.ProjectID(),
      Statuses:  []string{string(domain.RequirementStatusPreparing), string(domain.RequirementStatusCoding)},
  })
  if err != nil {
      return nil, fmt.Errorf("failed to count running requirements: %w", err)
  }
  if runningCount >= project.MaxConcurrentAgents() {
      return nil, ErrMaxConcurrentAgentsReached
  }
  ```

### 3. 后端接口层（Interfaces）

**文件**：`backend/interfaces/http/project_handler.go`

- `UpdateProjectRequest` 新增 `MaxConcurrentAgents *int`（`json:"max_concurrent_agents,omitempty"`）
- `UpdateProject` handler 将该字段传入 `UpdateProjectCommand`
- `projectToMap()` 增加 `"max_concurrent_agents": project.MaxConcurrentAgents()`

### 4. 后端基础设施层（Infrastructure）

**文件**：`backend/infrastructure/persistence/schema_tables.go`

- `projects` 表定义增加：
  ```sql
  max_concurrent_agents INTEGER NOT NULL DEFAULT 2,
  ```

**文件**：`backend/infrastructure/persistence/project_repository.go`

- `Save()`：INSERT/UPSERT SQL 增加 `max_concurrent_agents` 列和更新字段
- `FindByID()` / `FindAll()`：SELECT 增加 `max_concurrent_agents`
- `scanProject()`：解析该列并传入 `ProjectSnapshot`

**文件**：`backend/infrastructure/persistence/migrations.go`

- 新增 `MigrateMaxConcurrentAgentsColumn(db *sql.DB) error`
- 使用 `ALTER TABLE projects ADD COLUMN max_concurrent_agents INTEGER NOT NULL DEFAULT 2`

**文件**：`backend/cmd/server/main.go`

- 在现有迁移调用后新增 `MigrateMaxConcurrentAgentsColumn(db)`

### 5. 前端实现

**文件**：`frontend/src/types/projectRequirement.ts`

- `Project` 接口新增 `max_concurrent_agents: number`
- `UpdateProjectRequest` 接口新增 `max_concurrent_agents?: number`

**文件**：`frontend/src/pages/ProjectRequirementPage.tsx`

- "基本信息" Form 的 `initialValues` 增加 `max_concurrent_agents`
- 新增表单项：
  ```tsx
  <Form.Item label="最大并发 Agent 数" name="max_concurrent_agents" rules={[{ required: true, message: '请输入最大并发 Agent 数' }]}>
    <InputNumber min={1} max={10} style={{ width: 120 }} />
  </Form.Item>
  ```
- `onFinish` 中将 `values.max_concurrent_agents` 传入 `updateProject`
- 项目创建/编辑 Modal 的表单初始值和提交也同步该字段

## 数据流

```
前端配置页 ──UpdateProjectRequest──→ HTTP Handler ──UpdateProjectCommand──→ Project Service
                                                                          ↓
                                                                    domain.Project.SetMaxConcurrentAgents()
                                                                          ↓
                                                                    SQLiteProjectRepository.Save()
                                                                          ↓
                                                                      SQLite DB

需求派发请求 ──DispatchRequirementCommand──→ RequirementDispatchService
                                                  ↓
                                       requirementRepo.Count(preparing+coding)
                                                  ↓
                                       runningCount < max ? 继续派发 : ErrMaxConcurrentAgentsReached
```
