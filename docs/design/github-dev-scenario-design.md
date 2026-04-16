# GitHub 开发协作心跳场景设计文档

## 概述

本设计引入 **HeartbeatScenario（心跳场景）** 领域模型，将一组协同工作的心跳模板封装为可复用的场景。项目可以通过选择场景一键初始化对应的完整心跳集，实现从 GitHub Issue 到 PR 合并的自动化协作流水线。

## 数据模型

### 后端 Domain

#### HeartbeatScenarioID 值对象
```go
type HeartbeatScenarioID struct{ value string }
```

#### HeartbeatScenarioItem 值对象（场景中的单条心跳定义）
```go
type HeartbeatScenarioItem struct {
    name            string
    intervalMinutes int
    mdContent       string
    agentCode       string
    requirementType string
    sortOrder       int
}
```

#### HeartbeatScenario 聚合根
```go
type HeartbeatScenario struct {
    id          HeartbeatScenarioID
    code        string  // 唯一编码，如 "github_dev_workflow"
    name        string  // 展示名称
    description string
    items       []HeartbeatScenarioItem
    enabled     bool    // 是否可用
    isBuiltIn   bool    // 是否系统内置
    createdAt   time.Time
    updatedAt   time.Time
}
```

核心行为：
- `NewHeartbeatScenario(...)` — 创建场景，验证 code/name 非空。
- `Update(...)` — 更新名称、描述、items。
- `ApplyToProject(projectID ProjectID, idGen IDGenerator) ([]*Heartbeat, error)` — 将场景实例化为一组项目心跳。

### Project 扩展

在 `Project` 聚合根中新增字段：
```go
type Project struct {
    // ... 现有字段 ...
    heartbeatScenarioCode string
}
```

新增行为：
- `SetHeartbeatScenarioCode(code string)`
- `HeartbeatScenarioCode() string`

### 数据库表

#### 新增 heartbeat_scenarios 表
```sql
CREATE TABLE IF NOT EXISTS heartbeat_scenarios (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    items TEXT NOT NULL DEFAULT '[]',  -- JSON 数组
    enabled INTEGER NOT NULL DEFAULT 1,
    is_built_in INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
```

#### 修改 projects 表
```sql
ALTER TABLE projects ADD COLUMN heartbeat_scenario_code TEXT NOT NULL DEFAULT '';
```

## 架构设计

### 后端分层

| 层级 | 文件 | 职责 |
|------|------|------|
| Domain | `domain/heartbeat_scenario.go` | HeartbeatScenario 聚合根、HeartbeatScenarioItem 值对象 |
| Domain | `domain/heartbeat_scenario_repository.go` | 仓储接口 `HeartbeatScenarioRepository` |
| Infrastructure | `infrastructure/persistence/heartbeat_scenario_repository.go` | SQLite 实现 |
| Application | `application/heartbeat_scenario_service.go` | 场景 CRUD、应用场景到项目 |
| Interfaces | `interfaces/http/heartbeat_scenario_handler.go` | REST API Handler |
| CLI | `cmd/cli/cmd/project_scenario.go` | CLI 命令：`taskmanager project scenario ...` |

### API 设计

#### 场景管理
- `GET /api/v1/heartbeat-scenarios` — 列出所有场景
- `GET /api/v1/heartbeat-scenarios/:code` — 按编码获取场景详情

#### 项目场景绑定
- `POST /api/v1/projects/:project_id/apply-scenario` — 为项目应用场景
  - Body: `{ "scenario_code": "github_dev_workflow" }`
  - 行为：先删除该项目下所有由旧场景生成且未被手动修改的心跳，再创建新场景的心跳实例。

### 前端交互设计

在**项目编辑/配置页面**的"心跳管理"区域上方增加"心跳场景"选择器：

```
心跳场景: [ 请选择场景 ▼ ]
         [ 应用此场景 ]
```

- 下拉框加载 `/api/v1/heartbeat-scenarios` 列表。
- 选择场景后点击"应用"，调用 `POST /api/v1/projects/:project_id/apply-scenario`。
- 应用成功后刷新项目心跳列表，展示新注册的心跳。
- 场景应用后，每个心跳独立可编辑、启停。

## 内置场景：GitHub 开发协作工作流

### 预置编码
`github_dev_workflow`

### 预置心跳集

每个心跳的 Prompt 遵循统一模板：说明任务目标、约束条件（使用 `gh` CLI）、项目信息变量替换、以及"无待处理项则直接返回"的兜底规则。

#### 1. Issue 分析 (`github_issue_analyzer`)
- **interval**: 30
- **requirement_type**: `github_issue`
- **Prompt 要点**：
  - 使用 `gh issue list --repo owner/repo --state open` 获取 open issues。
  - 对每个未被评论过的 issue，clone 仓库到临时目录，结合代码库分析 issue 描述中的问题。
  - 将分析结论（问题根因、可能影响的文件、建议修复方向）以评论形式发布到该 issue 下。
  - 如果没有 open issues，直接返回"当前无待分析 issue"。

#### 2. LGTM 代码编写 (`github_lgtm_coder`)
- **interval**: 30
- **requirement_type**: `github_coding`
- **Prompt 要点**：
  - 使用 `gh issue list --repo owner/repo --label lgtm --state open` 获取已评审通过的 issue。
  - 对最旧的一个 issue：clone 仓库、创建 feature 分支、根据 issue 描述实现代码修改、运行基础测试（如有）、push 分支。
  - 使用 `gh pr create` 创建 PR，并在 PR 描述中通过 `Closes #issue_number` 关联 issue。
  - 如果没有带 `lgtm` 标签的 open issue，直接返回。

#### 3. PR 需求评审 (`github_pr_requirement_review`)
- **interval**: 30
- **requirement_type**: `github_pr_review`
- **Prompt 要点**：
  - 使用 `gh pr list --repo owner/repo --state open` 获取 open PRs。
  - 检查每个 PR 的评论中是否包含"需求评审通过"字样。
  - 若不存在，通过 PR 描述或 API 查找关联的 issue，阅读 issue 内容后在 PR 下评论：
    - 原始需求摘要
    - 需求评审结论（通过/需补充信息）
  - 如果所有 PR 都已通过需求评审，直接返回。

#### 4. PR 代码质量评审 (`github_pr_code_review`)
- **interval**: 30
- **requirement_type**: `github_pr_review`
- **Prompt 要点**：
  - 使用 `gh pr list --repo owner/repo --state open` 获取 open PRs。
  - 对未被自己评审过的 PR，使用 `gh pr view` 和 `gh pr diff` 查看变更。
  - 从代码质量、潜在 bug、安全漏洞、性能问题等角度进行评审，并将评审意见以评论形式发布到 PR 下。
  - 如果没有待评审 PR，直接返回。

#### 5. PR 修改修复 (`github_pr_fixer`)
- **interval**: 30
- **requirement_type**: `github_coding`
- **Prompt 要点**：
  - 使用 `gh pr list --repo owner/repo --state open --author @me`（或类似逻辑）查找自己创建的 open PR。
  - 阅读 PR 评论中的修改建议，判断哪些是可执行的合理建议。
  - checkout PR 分支、按建议修改代码、commit、push 更新。
  - 在 PR 下评论说明已修复的内容。
  - 如果没有待修复建议，直接返回。

#### 6. PR 合并检查 (`github_pr_merge_check`)
- **interval**: 30
- **requirement_type**: `github_pr_review`
- **Prompt 要点**：
  - 使用 `gh pr list --repo owner/repo --state open` 获取 open PRs。
  - 对每个 PR 检查：CI 是否通过（`gh pr checks`）、是否有至少一条 `/lgtm` 或类似批准评论、是否有未解决的修改建议。
  - 若满足合并条件且尚未被标记为可合并，在 PR 下评论 `/lgtm`。
  - 如果没有满足条件的 PR，直接返回。

#### 7. PR 文档补充 (`github_pr_doc_writer`)
- **interval**: 60
- **requirement_type**: `github_doc`
- **Prompt 要点**：
  - 使用 `gh pr list --repo owner/repo --state open` 获取近期有代码变更的 PR。
  - 查看 PR diff，判断是否需要更新 README、API 文档、变更日志等。
  - 若需要，在 PR 分支上补充文档（或创建文档更新 PR），并说明更新的内容。
  - 如果无需补充文档，直接返回。

#### 8. PR 测试补充 (`github_pr_test_writer`)
- **interval**: 60
- **requirement_type**: `github_test`
- **Prompt 要点**：
  - 使用 `gh pr list --repo owner/repo --state open` 获取 open PRs。
  - 查看 PR diff，识别新增/修改的功能点，判断是否需要补充单元测试、集成测试。
  - 若需要，在 PR 分支上编写并补充相关测试，运行测试确保通过，push 更新。
  - 如果所有 PR 测试都已充足，直接返回。

## 应用场景流程

```
用户选择场景 ──► 调用 ApplyScenario ──► 查找场景定义
                                          │
                                          ▼
                                   实例化 Heartbeat[]
                                          │
                                          ▼
                                   批量保存到 heartbeats 表
                                          │
                                          ▼
                                   Scheduler.RefreshSchedule
                                          │
                                          ▼
                                   心跳按周期自动触发
```

## 状态机建议（可选增强）

可为各需求类型绑定状态机，形成更严格的工作流：

| requirement_type | 建议状态流 |
|------------------|-----------|
| `github_issue` | open → analyzing → commented → done |
| `github_coding` | open → coding → pr_created → done |
| `github_pr_review` | open → reviewing → approved → done |
| `github_doc` | open → writing → committed → done |
| `github_test` | open → testing → committed → done |

本版本先实现基础心跳场景，状态机绑定由用户后续按需配置。
