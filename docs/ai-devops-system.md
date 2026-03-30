# AI DevOps 系统设计方案

## 一、系统愿景

**目标：构建一个 AI 原生的 DevOps 系统，让 AI 能够自主决策、自动执行软件开发全流程。**

```
人类：下达高层指令 "完成用户认证模块"
    ↓
AI 调度器：自主观察、分析、决策、行动
    ↓
Toolbox：执行原子操作
    ↓
Agent：具体执行代码
```

**核心能力：**
- AI 调度器能够观察当前状态
- AI 调度器能够决策下一步该做什么
- AI 调度器能够指挥调度 Agent 执行
- AI 调度器能够处理错误并恢复
- 人类只在必要时介入

## 二、AI 调度器（核心）

### 2.1 什么是 AI 调度器

AI 调度器是整个系统的"大脑"，负责：
- 持续监控项目状态
- 分析判断当前情况
- 决策下一步行动
- 指挥 Agent 执行

**类比：**
```
传统开发：人类项目经理 → 分配任务 → 监控进度 → 处理问题
AI 时代：AI 调度器   → 调度任务 → 监控进度 → 处理问题
```

### 2.2 调度器的数据来源

**必须有统一的"调度任务表"，这是调度器唯一的监管信息来源。**

```sql
-- 调度任务表（核心！）
CREATE TABLE schedule_tasks (
    id TEXT PRIMARY KEY,

    -- 基本信息
    name TEXT NOT NULL,                    -- 项目名称
    description TEXT,                      -- 描述

    -- 关联来源
    source_type TEXT NOT NULL,             -- requirement, task, custom
    source_id TEXT NOT NULL,               -- 对应的 ID

    -- 当前状态
    status TEXT NOT NULL DEFAULT 'pending',  -- pending, running, paused, completed, failed
    current_stage TEXT,                    -- 当前阶段
    current_command TEXT,                  -- 当前下达的指令

    -- 处理信息
    assignee TEXT,                          -- 负责人
    handler TEXT,                          -- 当前处理者

    -- 调度循环信息
    loop_count INTEGER DEFAULT 0,          -- 循环次数
    last_loop_at INTEGER,                  -- 上次循环时间
    next_action TEXT,                      -- 下一步行动

    -- 结果
    result TEXT,
    completed_at INTEGER,

    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
```

### 2.3 调度器的工作方式

```
┌─────────────────────────────────────────────────────────────┐
│                   AI 调度器工作方式                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. 从 schedule_tasks 读取 status='running' 的任务         │
│  2. 对每个任务执行 OODA 循环                              │
│  3. 更新任务的 stage, next_action 等字段                    │
│  4. 下达指令到 command_records                            │
│  5. 循环直到任务完成/暂停                                  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## 三、OODA 循环（调度器的核心算法）

### 3.1 循环图

```
┌─────────────────────────────────────────────────────────────┐
│                    AI 调度器 OODA 循环                      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   ┌─────────┐                                              │
│   │ Observe │ ← 从 schedule_tasks 读取状态                   │
│   └────┬────┘    + 调用 Toolbox 观察工具                  │
│        │                                                    │
│        ▼                                                    │
│   ┌─────────┐                                              │
│   │  Orient │ ← 分析：任务失败？完成？阻塞？                │
│   └────┬────┘                                              │
│        │                                                    │
│        ▼                                                    │
│   ┌─────────┐                                              │
│   │  Decide │ ← 决策：修复？继续？通知人？                  │
│   └────┬────┘                                              │
│        │                                                    │
│        ▼                                                    │
│   ┌─────────┐                                              │
│   │   Act   │ ← 下达指令 / 通知人类                        │
│   └────┬────┘                                              │
│        │                                                    │
│        └────────────── 再观察 ◀───────────────────────────│
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 调度器代码

```go
type Scheduler struct {
    store    *Store
    toolbox  *Toolbox
    executor *Executor
}

func (s *Scheduler) Run() {
    // 定时检查（每 30 秒）
    ticker := time.NewTicker(30 * time.Second)

    for range ticker.C {
        // 获取所有需要调度的任务
        tasks, _ := s.store.GetActiveTasks()

        for _, task := range tasks {
            s.OODALoop(task)
        }
    }
}

func (s *Scheduler) OODALoop(task *ScheduleTask) {
    for {
        // 1. Observe - 观察状态
        state := s.Observe(task)

        // 2. Orient - 分析
        analysis := s.Analyze(state)

        // 3. Decide - 决策
        decision := s.Decide(analysis, task)

        // 4. Act - 行动
        result := s.Act(decision, task)

        // 更新任务状态
        s.updateTask(task, decision)

        // 结束条件
        if decision.Done {
            break
        }

        // 等待下一个触发（事件或定时）
        s.waitForTrigger()
    }
}
```

### 3.3 决策逻辑

```go
func (s *Scheduler) Decide(analysis Analysis, task *ScheduleTask) Decision {
    switch analysis.Type {

    case "task_failed", "test_failed":
        // 失败，尝试修复
        fixCount := s.store.GetFixCount(analysis.Target)
        if fixCount >= 3 {
            return Decision{
                Type:   "notify_human",
                Reason: "多次修复失败，需要人工介入",
                Done:   false,
            }
        }
        return Decision{
            Type:   "fix_issues",
            Reason: "尝试修复问题",
            Done:   false,
        }

    case "blocked":
        // 阻塞，通知人类
        return Decision{
            Type:   "notify_human",
            Reason: analysis.Reason,
            Done:   false,
        }

    case "all_completed":
        // 全部完成，通知验收
        return Decision{
            Type:   "notify_human",
            Reason: "所有任务完成，请验收",
            Done:   true,
        }

    case "in_progress":
        // 进行中，继续调度
        return Decision{
            Type:   "continue",
            Reason: analysis.Reason,
            Done:   false,
        }
    }

    return Decision{Done: true}
}
```

## 四、触发机制

### 4.1 三种触发方式

| 触发方式 | 时机 | 说明 |
|---------|------|------|
| **事件触发** | Agent 完成/失败时 | 立即检查，实时响应 |
| **定时触发** | 每 30 秒 | 检查孤儿任务（兜底） |
| **手动触发** | 人类命令 | 按需检查 |

### 4.2 事件总线

```go
type EventBus struct {
    subscribers map[string][]chan Event
}

type Event struct {
    Type          string
    TaskID        string
    RequirementID string
    Data          map[string]interface{}
    Timestamp     time.Time
}

func (eb *EventBus) Publish(e Event) {
    for _, ch := range eb.subscribers[e.Type] {
        select {
        case ch <- e:
        default:
        }
    }
}
```

**事件类型：**

| 事件 | 触发 | 调度器动作 |
|------|------|-----------|
| task_completed | Agent 报告完成 | 检查下游任务 |
| task_failed | Agent 报告失败 | 尝试修复 |
| human_approved | 人类审批通过 | 继续执行 |

## 五、Toolbox（工具箱）

### 5.1 工具分类

```
┌─────────────────────────────────────────────────────────────┐
│                      Toolbox（工具箱）                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  观察类（Observe）                                          │
│  ├── get_task_status()           查看任务状态                │
│  ├── get_test_results()          查看测试结果                │
│  ├── get_build_logs()           查看构建日志                │
│  └── get_timeout_tasks()        查看超时任务                │
│                                                             │
│  分析类（Analyze）                                          │
│  ├── analyze_error_pattern()      分析错误模式                │
│  └── check_dependency_health()    检查依赖健康度              │
│                                                             │
│  执行类（Act）                                              │
│  ├── analyze_requirement()       分析需求                    │
│  ├── split_tasks()              拆解任务                    │
│  ├── develop_code()             开发代码                    │
│  ├── run_tests()                运行测试                    │
│  ├── fix_issues()              修复问题                    │
│  └── deploy()                   部署                        │
│                                                             │
│  通知类（Notify）                                           │
│  ├── notify_human()            通知人类                    │
│  └── send_message()            发送消息                    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 5.2 Tool 接口

```go
type Tool interface {
    Name() string
    Execute(params map[string]interface{}) (Result, error)
}

type Toolbox struct {
    tools map[string]Tool
}

func (tb *Toolbox) Execute(toolName string, params map[string]interface{}) (Result, error) {
    tool, ok := tb.tools[toolName]
    if !ok {
        return Result{}, fmt.Errorf("tool not found: %s", toolName)
    }
    return tool.Execute(params)
}
```

## 六、指令记录

### 6.1 command_records（指令记录表）

```sql
CREATE TABLE command_records (
    id TEXT PRIMARY KEY,
    schedule_task_id TEXT NOT NULL,     -- 关联的调度任务

    -- 指令信息
    command TEXT NOT NULL,             -- 指令类型
    command_params TEXT,               -- 指令参数 (JSON)
    instruction TEXT,                  -- 具体指令内容

    -- 执行状态
    status TEXT NOT NULL DEFAULT 'pending',  -- pending, executing, completed, failed
    result TEXT,                      -- 执行结果
    error TEXT,                       -- 错误信息

    executor TEXT,                    -- 执行者
    started_at INTEGER,               -- 开始时间
    completed_at INTEGER,             -- 完成时间

    created_at INTEGER NOT NULL,
    FOREIGN KEY (schedule_task_id) REFERENCES schedule_tasks(id)
);
```

### 6.2 指令流程

```
调度器决策下达指令
    ↓
写入 command_records (status='pending')
    ↓
Agent 领取指令 (status='executing')
    ↓
Agent 执行完成 (status='completed' 或 'failed')
    ↓
触发下一次 OODA 循环
```

## 七、人类介入

### 7.1 介入条件

| 情况 | 触发条件 | 通知内容 |
|------|---------|---------|
| 设计评审 | 分析完成 | "设计文档已生成，请评审" |
| 任务阻塞 | 依赖失败 | "Task XXX 失败，阻塞后续" |
| 修复超时 | 同一问题修复 3 次仍失败 | "需要人工介入" |
| 最终验收 | 所有任务完成 | "开发完成，请验收" |

### 7.2 介入方式

人类通过飞书通知收到消息后：
- 点击链接查看详情
- 审批通过/驳回
- 直接在飞书上下达新指令

## 八、完整数据流

```
┌─────────────────────────────────────────────────────────────┐
│                     完整数据流                              │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. schedule_tasks（调度任务表）                           │
│     - 统一存储所有需要调度的项目                           │
│     - 记录当前阶段、下一步行动                             │
│                                                             │
│  2. 调度器主循环（定时 30 秒）                           │
│     - 读取 status='running' 的任务                         │
│     - 执行 OODA 循环                                       │
│                                                             │
│  3. OODA 循环                                             │
│     Observe → Orient → Decide → Act                       │
│                                                             │
│  4. Act 下达指令                                          │
│     - 写入 command_records                                 │
│     - 更新 schedule_tasks.current_command                  │
│                                                             │
│  5. Agent 执行完成                                         │
│     - 更新 command_records.status                          │
│     - 发布事件 (task_completed / task_failed)             │
│                                                             │
│  6. 事件触发                                              │
│     - 调度器收到事件，再次执行 OODA 循环                   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## 九、数据模型

### 9.1 任务记录表

```sql
CREATE TABLE task_records (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL,
    stage TEXT NOT NULL DEFAULT 'created',
    status TEXT NOT NULL DEFAULT 'pending',
    percent INTEGER DEFAULT 0,
    acceptance_criteria TEXT,
    assignee TEXT,
    confirmed_by TEXT,
    confirmed_at INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
```

### 9.2 阶段定义

| Stage | 说明 |
|-------|------|
| created | 创建 |
| analyzing | 分析中 |
| designing | 设计中 |
| developed | 开发完成 |
| testing | 测试中 |
| verified | 测试通过 |
| deployed | 已部署 |
| accepted | 已验收 |

## 十、实施计划

### Phase 1: MVP
1. 实现 schedule_tasks 表 + CRUD
2. 实现 command_records 表
3. 实现 OODA 循环
4. 实现基础的 Toolbox
5. 手动触发，观察执行

### Phase 2: 自动调度
1. 实现事件总线
2. 实现 Agent 执行器
3. 实现定时检查
4. 实现错误恢复

### Phase 3: 完善
1. 飞书集成
2. 更多工具
3. 并行执行

## 十一、最小化 MVP：手动触发 Agent 开发并发起 PR

### 11.1 目标范围

本 MVP 只覆盖一条最短可用链路：

1. 维护项目基本信息（Git 仓库地址、初始化步骤）
2. 维护项目需求列表
3. 人工选定一个需求并手动触发指定 Agent 执行
4. 为该需求创建临时工作目录
5. 基于 CodingAgent 生成分身 Agent（仅工作目录不同）
6. 分身 Agent 在临时目录完成环境初始化、实现需求、发起 PR
7. 回写需求开发状态与 PR 信息

### 11.2 核心对象

#### Project（项目）

```json
{
  "id": "proj_xxx",
  "name": "任务平台",
  "git_repo_url": "git@github.com:org/repo.git",
  "default_branch": "main",
  "init_steps": [
    "go mod tidy",
    "make test"
  ]
}
```

#### Requirement（需求）

```json
{
  "id": "req_xxx",
  "project_id": "proj_xxx",
  "title": "支持任务筛选",
  "description": "任务列表支持按状态筛选",
  "status": "todo",
  "dev_state": "idle",
  "assignee_agent_id": "",
  "workspace_path": "",
  "branch_name": "",
  "pr_url": ""
}
```

建议状态字段：

- `status`：`todo` / `in_progress` / `done`
- `dev_state`：`idle` / `preparing` / `coding` / `pr_opened` / `failed`

#### AgentReplica（Agent 分身）

```json
{
  "id": "agent_replica_xxx",
  "base_agent_id": "coding_agent_default",
  "type": "coding_agent_replica",
  "workdir": "/tmp/ai-devops/proj_xxx/req_xxx",
  "requirement_id": "req_xxx",
  "status": "running"
}
```

### 11.3 手动触发流程

```text
[人类] 在需求列表选择 req_xxx，点击“开始开发”
   ↓
[调度器] 校验项目信息与需求状态（必须是 todo/idle）
   ↓
[调度器] 创建临时工作目录 /tmp/ai-devops/{project_id}/{requirement_id}
   ↓
[调度器] 基于 CodingAgent 创建分身，覆盖 workdir=临时目录
   ↓
[调度器] 下达任务给分身 Agent：
         1) clone 仓库并切分支
         2) 执行项目 init_steps
         3) 实现需求并本地自检
         4) 提交代码并推送分支
         5) 发起 PR
   ↓
[分身 Agent] 回传执行结果（成功/失败、PR 链接、错误日志）
   ↓
[调度器] 更新需求状态与开发状态
```

### 11.4 调度器执行协议（MVP）

#### 输入参数

```json
{
  "project_id": "proj_xxx",
  "requirement_id": "req_xxx",
  "trigger_by": "human_user_id",
  "agent_id": "coding_agent_default"
}
```

#### 调度动作

1. 读取 `Project` 与 `Requirement`
2. 原子更新需求状态：`status=todo, dev_state=idle` -> `status=in_progress, dev_state=preparing`
3. 创建 workspace 目录
4. 创建 Agent 分身并绑定 requirement
5. 发送执行指令（包含 repo、init_steps、需求描述、验收标准）
6. 监听执行事件并写回 requirement 状态

#### 输出结果

```json
{
  "requirement_id": "req_xxx",
  "status": "in_progress",
  "dev_state": "coding",
  "workspace_path": "/tmp/ai-devops/proj_xxx/req_xxx",
  "replica_agent_id": "agent_replica_xxx"
}
```

### 11.5 分身 Agent 标准执行步骤

1. 进入 `workdir`
2. `git clone <git_repo_url> .`
3. `git checkout -b feature/req_xxx`
4. 逐条执行 `init_steps`
5. 根据需求实现代码
6. 运行最小自检（例如单测、lint）
7. `git add/commit/push`
8. 调用 Git 平台 API 发起 PR
9. 回传 `pr_url` 与执行摘要

### 11.6 状态回写规则

#### 成功

- `status`: `done`
- `dev_state`: `pr_opened`
- `pr_url`: 非空

#### 失败

- `status`: 保持 `in_progress`（等待重试或人工处理）
- `dev_state`: `failed`
- 记录 `last_error`

### 11.7 最小表结构补充

```sql
-- 项目表
CREATE TABLE projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    git_repo_url TEXT NOT NULL,
    default_branch TEXT NOT NULL DEFAULT 'main',
    init_steps TEXT NOT NULL, -- JSON 数组
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- 需求表
CREATE TABLE requirements (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'todo',
    dev_state TEXT NOT NULL DEFAULT 'idle',
    assignee_agent_id TEXT,
    workspace_path TEXT,
    branch_name TEXT,
    pr_url TEXT,
    last_error TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id)
);
```

### 11.8 MVP 验收标准

满足以下条件即视为 MVP 完成：

1. 能创建项目并保存 Git 仓库地址、初始化步骤
2. 能创建需求并在列表展示状态
3. 能手动触发某需求进入开发流程
4. 触发后会创建独立临时工作目录
5. 使用 CodingAgent 分身在该目录执行初始化和开发
6. 需求完成后自动发起 PR 并写回 `pr_url`
7. 需求状态能正确更新为 `done/pr_opened` 或 `failed`

## 十二、MVP 落地设计（复用现有架构，指导 AI 前后端开发）

### 12.1 设计目标

本章用于指导 AI 在当前代码体系内完成 MVP 开发，要求：

1. 不推翻现有 DDD 分层与模块边界
2. 复用已有 Agent 管理、Task 执行、HTTP 接口与前端管理台模式
3. 以“最小可用”为优先，实现端到端闭环：创建项目 -> 创建需求 -> 手动触发 -> Agent 分身执行 -> PR 回写

### 12.2 架构复用原则

#### 后端复用点

- **Interface 层**：沿用 `backend/interfaces/http` 的 handler + router 组织方式
- **Application 层**：沿用命令对象 + `ApplicationService` 编排方式
- **Domain 层**：沿用聚合根 + 仓储接口（定义在 `domain/repository.go`）
- **Infrastructure 层**：沿用 SQLite 仓储实现与 `schema.go` 统一迁移入口

#### 前端复用点

- 沿用 `frontend/src/api/*Api.ts` 的 API 封装风格（`apiClient` + Bearer Token）
- 沿用 React + Ant Design 页面组织方式
- 沿用“页面 + 组件 + hooks”模式（参考 AgentManagement）
- 路由统一挂载在 `frontend/src/App.tsx`

### 12.3 模块拆分设计

#### 12.3.1 后端新增模块

建议新增以下模块，命名与现有风格保持一致：

1. `ProjectApplicationService`（项目 CRUD）
2. `RequirementApplicationService`（需求 CRUD + 状态更新）
3. `RequirementDispatchService`（手动触发开发，创建分身并下发执行）

建议新增实体（聚合）：

1. `Project`
2. `Requirement`
3. `RequirementExecution`（可选，MVP 可并入 Requirement 字段）

#### 12.3.2 前端新增模块

建议新增页面：

1. `ProjectRequirementPage`（项目与需求管理）

建议新增组件：

1. `ProjectForm`
2. `RequirementList`
3. `RequirementDispatchModal`

建议新增 API 客户端：

1. `projectApi.ts`
2. `requirementApi.ts`

### 12.4 数据模型设计（与现有 SQLite 兼容）

#### 12.4.1 projects 表

```sql
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    git_repo_url TEXT NOT NULL,
    default_branch TEXT NOT NULL DEFAULT 'main',
    init_steps TEXT NOT NULL, -- JSON 数组
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_projects_name ON projects(name);
```

#### 12.4.2 requirements 表

```sql
CREATE TABLE IF NOT EXISTS requirements (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    acceptance_criteria TEXT,
    status TEXT NOT NULL DEFAULT 'todo',
    dev_state TEXT NOT NULL DEFAULT 'idle',
    assignee_agent_id TEXT,
    replica_agent_id TEXT,
    workspace_path TEXT,
    branch_name TEXT,
    pr_url TEXT,
    last_error TEXT,
    started_at INTEGER,
    completed_at INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE INDEX IF NOT EXISTS idx_requirements_project_id ON requirements(project_id);
CREATE INDEX IF NOT EXISTS idx_requirements_status ON requirements(status, dev_state);
```

### 12.5 状态机设计

#### 12.5.1 需求业务状态（status）

- `todo`：待开发
- `in_progress`：开发中
- `done`：开发完成（已发 PR）

#### 12.5.2 开发执行状态（dev_state）

- `idle`：尚未触发
- `preparing`：准备环境/创建分身
- `coding`：分身执行开发
- `pr_opened`：PR 已创建
- `failed`：执行失败

#### 12.5.3 状态约束

1. 仅 `todo + idle` 可被手动触发
2. 触发时原子更新为 `in_progress + preparing`
3. 分身进入执行后更新为 `in_progress + coding`
4. PR 成功后更新为 `done + pr_opened`
5. 失败时更新为 `in_progress + failed`

### 12.6 后端接口设计（遵循 /api/v1 规范）

#### 12.6.1 项目接口

- `POST /api/v1/projects`：创建项目
- `GET /api/v1/projects`：列表或按 `id` 查询
- `PUT /api/v1/projects?id={id}`：更新项目
- `DELETE /api/v1/projects?id={id}`：删除项目

#### 12.6.2 需求接口

- `POST /api/v1/requirements`：创建需求
- `GET /api/v1/requirements?project_id={projectID}`：需求列表
- `PUT /api/v1/requirements?id={id}`：更新需求内容

#### 12.6.3 手动触发接口

- `POST /api/v1/requirements/{id}/dispatch`

请求体：

```json
{
  "agent_id": "coding_agent_default"
}
```

响应体：

```json
{
  "requirement_id": "req_xxx",
  "status": "in_progress",
  "dev_state": "coding",
  "workspace_path": "/tmp/ai-devops/proj_xxx/req_xxx",
  "replica_agent_id": "agent_replica_xxx"
}
```

### 12.7 后端实现分层指引（给 AI 的编码约束）

#### 12.7.1 Domain 层

1. 在 `domain` 新增 `project.go`、`requirement.go`
2. 新增仓储接口：`ProjectRepository`、`RequirementRepository`
3. 将状态转换规则收敛到 `Requirement` 方法中，例如：
   - `StartDispatch()`
   - `MarkCoding(workspace, replicaAgentID string)`
   - `MarkPROpened(prURL, branch string)`
   - `MarkFailed(errMsg string)`

#### 12.7.2 Application 层

1. `ProjectApplicationService`：负责项目 CRUD 编排
2. `RequirementApplicationService`：负责需求 CRUD 与查询
3. `RequirementDispatchService`：负责编排触发流程
   - 加载 requirement/project/base agent
   - 创建 workspace 路径
   - 基于 base agent 构建分身 agent（复用 agent 配置）
   - 覆盖分身 `ClaudeCodeConfig.Cwd`
   - 下发执行任务（可复用现有 task 创建与执行链路）
   - 按执行回调更新 requirement 状态

#### 12.7.3 Infrastructure 层

1. 扩展 `infrastructure/persistence/schema.go` 增加新表
2. 实现 `project_repository.go` 与 `requirement_repository.go`
3. 使用与现有仓储一致的序列化和错误处理模式

#### 12.7.4 Interface 层

1. 新增 `project_handler.go`、`requirement_handler.go`
2. 在 `router.go` 挂载 `/api/v1/projects` 与 `/api/v1/requirements` 路由
3. 复用现有 HTTP 错误响应结构：`{ code, message }`

### 12.8 Agent 分身策略（复用现有 Agent 模型）

MVP 不新增独立“分身表”，直接复用现有 `agents`：

1. 基于目标 CodingAgent 读取快照（模型、提示词、工具、技能、provider）
2. 创建新 agent，命名约定：`{base_name}-replica-{requirement_id}`
3. `agent_type` 仍为 `CodingAgent`
4. 复制原 `ClaudeCodeConfig`，仅覆盖：
   - `cwd=/tmp/ai-devops/{project_id}/{requirement_id}`
   - `continue_conversation=false`
   - `fork_session=true`（可选）
5. 将 `replica_agent_id` 回写 requirement

### 12.9 调度执行时序（手动触发）

```text
Frontend 点击“开始开发”
  -> POST /api/v1/requirements/{id}/dispatch
  -> RequirementDispatchService.Start()
  -> Requirement.StartDispatch()
  -> 创建 workspace 目录
  -> 基于 CodingAgent 创建 replica agent 并设置 cwd
  -> 创建/启动 Agent 类型任务（上下文绑定 replica_agent_code）
  -> 执行完成后更新 requirement:
       成功 -> MarkPROpened(pr_url, branch)
       失败 -> MarkFailed(last_error)
```

### 12.10 前端设计（复用现有管理台交互模式）

#### 12.10.1 页面结构

新增左侧菜单：`项目需求`

页面分区：

1. 上半区：项目信息管理（仓库地址、默认分支、初始化步骤）
2. 下半区：需求列表（状态、负责人、PR、操作）

#### 12.10.2 需求列表操作

每条需求提供操作按钮：

1. `开始开发`（仅 `todo/idle` 显示）
2. `查看 PR`（`pr_url` 非空显示）
3. `查看错误`（`failed` 显示）

#### 12.10.3 UI 状态映射

- `todo/idle`：灰色标签“待开发”
- `in_progress/preparing`：蓝色标签“准备中”
- `in_progress/coding`：处理中标签“开发中”
- `done/pr_opened`：绿色标签“PR 已创建”
- `in_progress/failed`：红色标签“失败”

### 12.11 AI 开发任务拆解（可直接执行）

#### 任务 A：后端 Domain + Repository

1. 新增 `Project` 与 `Requirement` 领域实体
2. 扩展 `domain/repository.go` 接口定义
3. 新增 SQLite 仓储实现与单元测试

#### 任务 B：后端 Application + HTTP

1. 新增项目/需求应用服务
2. 新增 dispatch 应用服务
3. 新增 handler 与 router 注册
4. 新增接口测试（重点覆盖 dispatch 状态流转）

#### 任务 C：前端 API + 页面

1. 新增 `projectApi.ts`、`requirementApi.ts`
2. 新增 `ProjectRequirementPage.tsx`
3. 在 `App.tsx` 注册菜单与路由
4. 增加 dispatch 交互与状态展示

#### 任务 D：联调与验收

1. 创建项目与需求测试数据
2. 触发 dispatch，检查 workspace 与分身创建
3. 模拟成功/失败回调，验证需求状态变化
4. 验证 PR 链接展示与跳转

### 12.12 验收清单（实现完成判定）

1. 后端可完成项目与需求 CRUD
2. `POST /api/v1/requirements/{id}/dispatch` 可用且有幂等保护
3. 分身 Agent 的 `cwd` 为需求独立目录
4. 需求状态按状态机正确流转
5. 前端可完成创建/列表/触发/状态展示
6. 成功场景可展示 PR 链接，失败场景可展示错误信息
7. 实现严格复用现有 DDD 分层与现有路由、API、页面组织方式
