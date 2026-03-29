# AI 开发平台 - 需求·任务·云盘 系统

## 1. 系统概述

### 1.1 核心理念

不再人工拆解任务，而是让 Coding Agent 自主完成"需求→开发→产物"的完整链路。

- **人**：负责提需求、看进度、用产物
- **AI**：负责理解需求、执行开发、提交产物

### 1.2 三要素

| 要素 | 定位 | 核心操作 |
|------|------|----------|
| 需求列表 | 要做什么 | 创建、启动、暂停、完成 |
| 任务看板 | 做到哪了 | 状态流转、进度更新 |
| 产物云盘 | 产出什么 | 上传、关联、浏览 |

---

## 2. 数据模型

### 2.1 需求 (Requirement)

```go
type Requirement struct {
    ID          string         // 唯一标识
    Title       string         // 需求标题
    Description string         // 详细描述（支持 Markdown）
    Status      RequirementStatus // 状态

    // 关联
    TaskID      *string        // 关联的任务 ID（可选）
    Artifacts   []string       // 关联的产物 ID 列表

    // Agent 执行信息
    AgentSession string        // Agent 会话 ID（用于恢复）
    AgentConfig  string        // Agent 配置快照

    // 元数据
    CreatedBy   string         // 创建人
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type RequirementStatus string
const (
    StatusDraft     RequirementStatus = "draft"      // 草稿
    StatusPending   RequirementStatus = "pending"    // 待启动
    StatusRunning   RequirementStatus = "running"    // 开发中
    StatusPaused    RequirementStatus = "paused"     // 暂停
    StatusCompleted RequirementStatus = "completed"  // 已完成
    StatusFailed    RequirementStatus = "failed"    // 失败
)
```

### 2.2 任务 (Task)

```go
type Task struct {
    ID          string     // 唯一标识
    RequirementID string   // 所属需求 ID

    Title       string     // 任务标题
    Description string     // 任务描述
    Status      TaskStatus // 状态

    // 看板位置
    Column      string     // 看板列：backlog | in_progress | review | done
    Position    int        // 列内顺序

    // 进度
    Progress    int        // 进度百分比 0-100
    Log         string     // 开发日志（Agent 实时更新）

    // 产物关联
    Artifacts   []string   // 关联的产物 ID

    // 元数据
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type TaskStatus string
const (
    TaskBacklog   TaskStatus = "backlog"
    TaskInProgress TaskStatus = "in_progress"
    TaskReview    TaskStatus = "review"
    TaskDone      TaskStatus = "done"
)
```

### 2.3 产物 (Artifact)

```go
type Artifact struct {
    ID          string       // 唯一标识
    Name        string       // 文件名
    Type        ArtifactType // 类型

    // 存储
    StorageType string       // local | oss | s3
    Path        string       // 存储路径/URL
    Size        int64        // 大小

    // 关联
    TaskID      *string      // 关联的任务 ID
    Tags        []string     // 标签

    // 元数据
    CreatedBy   string       // 创建人/Agent
    CreatedAt   time.Time
}

type ArtifactType string
const (
    ArtifactDoc    ArtifactType = "doc"      // 文档
    ArtifactCode   ArtifactType = "code"     // 代码
    ArtifactImage  ArtifactType = "image"    // 图片
    ArtifactOther  ArtifactType = "other"    // 其他
)
```

---

## 3. 工作流

### 3.1 需求生命周期

```
创建需求
    ↓
[草稿] ──→ [待启动] ──→ [开发中] ──→ [已完成]
                ↑              ↓
                └─── [暂停] ←──┘
                         ↓
                      [失败]
```

### 3.2 任务看板流转

```
┌──────────┐    开始    ┌───────────┐   完成    ┌────────┐
│  Backlog │ ───────→ │ In_Progress│ ───────→ │  Done  │
│  待处理   │           │   开发中   │          │  已完成  │
└──────────┘           └───────────┘          └────────┘
                              ↓
                         ┌────────┐
                         │ Review │
                         │  待审核  │
                         └────────┘
```

### 3.3 完整开发流程

```
1. 用户创建需求 (Requirement)
      ↓
2. 用户点击"启动" → 创建 Task (Backlog)
      ↓
3. Coding Agent 接收需求，开始开发
      ↓
4. Agent 实时更新 Task.Progress 和 Task.Log
      ↓
5. Agent 完成后，提交产物到云盘
      ↓
6. Task 流转到 Done，Requirement 标记 Completed
      ↓
7. 产物自动关联到 Requirement
```

---

## 4. 核心功能

### 4.1 需求列表

| 功能 | 描述 |
|------|------|
| 创建需求 | 输入标题、描述 |
| 启动开发 | 创建 Task，触发 Agent 执行 |
| 暂停/恢复 | 暂停 Agent 执行，保留现场 |
| 查看进度 | 看 Task 看板了解进展 |
| 关联产物 | 查看云盘中的产出物 |

### 4.2 任务看板

| 功能 | 描述 |
|------|------|
| 拖拽排序 | 在列内调整顺序 |
| 状态流转 | 拖拽到其他列改变状态 |
| 实时日志 | 看 Agent 开发过程中的日志 |
| 进度百分比 | Agent 自动更新 0-100% |
| 产物卡片 | 显示关联的产出物 |

### 4.3 产物云盘

| 功能 | 描述 |
|------|------|
| 上传产物 | Agent/用户 上传文件 |
| 分类浏览 | 按文档/代码/图片分类 |
| 标签管理 | 给产物打标签 |
| 关联任务 | 产物关联到具体 Task |
| 预览 | 支持 Markdown/图片/代码 预览 |

---

## 5. Coding Agent 集成

### 5.1 Agent 如何获取需求信息

```
用户: 帮我实现用户登录功能
      ↓
System: 创建 Requirement (title="用户登录功能")
      ↓
Agent 启动时，接收:
  - requirement.description = "用户登录功能..."
  - task.id = "task_xxx"
      ↓
Agent 可通过工具调用:
  - get_requirement(task_id) → 获取完整需求
  - update_progress(task_id, progress, log) → 更新进度
  - submit_artifact(task_id, file) → 提交产物
```

### 5.2 持久化机制

| 问题 | 解决方案 |
|------|----------|
| Agent 崩溃恢复 | 存储 `requirement.agent_session`，重启后恢复 |
| 进度丢失 | `task.progress` + `task.log` 实时持久化 |
| 产物丢失 | `artifact` 立即持久化到云盘 |

---

## 6. 数据库表设计

```sql
-- 需求表
CREATE TABLE requirements (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'draft',
    task_id TEXT,
    agent_session TEXT,
    agent_config TEXT,
    created_by TEXT,
    created_at INTEGER,
    updated_at INTEGER
);

-- 任务表
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    requirement_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'backlog',
    column_name TEXT NOT NULL DEFAULT 'backlog',
    position INTEGER NOT NULL DEFAULT 0,
    progress INTEGER NOT NULL DEFAULT 0,
    log TEXT,
    created_at INTEGER,
    updated_at INTEGER,
    FOREIGN KEY (requirement_id) REFERENCES requirements(id)
);

-- 产物表
CREATE TABLE artifacts (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    storage_type TEXT NOT NULL DEFAULT 'local',
    path TEXT NOT NULL,
    size INTEGER,
    task_id TEXT,
    tags TEXT,
    created_by TEXT,
    created_at INTEGER,
    FOREIGN KEY (task_id) REFERENCES tasks(id)
);

-- 需求-产物关联表
CREATE TABLE requirement_artifacts (
    requirement_id TEXT,
    artifact_id TEXT,
    PRIMARY KEY (requirement_id, artifact_id),
    FOREIGN KEY (requirement_id) REFERENCES requirements(id),
    FOREIGN KEY (artifact_id) REFERENCES artifacts(id)
);
```

---

## 7. 待讨论问题

### 7.1 云盘存储

| 问题 | 选项 | 决策 |
|------|------|------|
| 存储位置 | A) 本地 (./artifacts)  B) OSS  C) S3  D) 混合 | __ |
| 单文件大小限制 | A) 10MB  B) 50MB  C) 100MB  D) 无限制 | __ |
| 文件清理机制 | A) 手动清理  B) 7天后自动清理  C) 按需求关联清理  D) 不清理 | __ |
| 最大存储总量 | A) 1GB  B) 10GB  C) 100GB  D) 无限制 | __ |

### 7.2 Agent 中断恢复

| 问题 | 选项 | 决策 |
|------|------|------|
| 会话保留时长 | A) 10分钟  B) 30分钟  C) 1小时  D) 24小时  E) 永久保留 | __ |
| 超时后处理 | A) 重新开始  B) 提示用户  C) 自动尝试恢复  D) 保存现场等待用户决策 | __ |
| Agent 失控处理 | A) 用户手动终止  B) 超时自动终止 (如 2小时)  C) 连续错误自动暂停 | __ |
| 进度更新频率 | A) 每分钟  B) 每5分钟  C) Agent 主动更新  D) 每个关键节点 | __ |

### 7.3 权限控制

| 问题 | 选项 | 决策 |
|------|------|------|
| 多用户支持 | A) 单用户  B) 基础多用户 (无隔离)  C) 完整多租户 | __ |
| 产物访问控制 | A) 公开  B) 按需求隔离  C) 按用户隔离 | __ |
| 需求操作权限 | A) 所有人可创建/编辑  B) 创建者可编辑  C) 需审批 | __ |
| 审计日志 | A) 不需要  B) 关键操作记录  C) 完整操作日志 | __ |

### 7.4 任务与看板

| 问题 | 选项 | 决策 |
|------|------|------|
| Task 依赖 | A) 不支持依赖  B) 支持前后依赖  C) 支持并行分支 | __ |
| 并行任务数 | A) 单任务串行  B) 同需求最多2个并行  C) 不限制 | __ |
| 看板列设置 | A) 固定4列 (Backlog/InProgress/Review/Done)  B) 可自定义列  C) 简化为3列 | __ |
| 任务创建时机 | A) 需求启动时自动创建  B) 用户手动创建  C) Agent 动态创建 | __ |

### 7.5 产物管理

| 问题 | 选项 | 决策 |
|------|------|------|
| 产物预览 | A) 仅下载  B) 支持 Markdown/图片预览  C) 支持代码高亮  D) 全部支持 | __ |
| 代码产物处理 | A) 压缩包存储  B) 仓库形式存储  C) 直接展示代码片段 | __ |
| 产物版本 | A) 不支持版本  B) 覆盖存储  C) 保留历史版本 | __ |

### 7.6 与现有系统

| 问题 | 选项 | 决策 |
|------|------|------|
| 是否复用现有 TASK 表 | A) 复用  B) 新建表  C) 迁移数据后新建 | __ |
| 是否复用 Agent 配置 | A) 复用现有 Agent  B) 独立配置  C) 统一配置中心 | __ |
| 产物仓库形式 | A) 存放在项目目录  B) 独立 artifacts 目录  C) Git 仓库形式 | __ |

---

## 8. UI 原型（待设计）

### 8.1 需求列表页
- 列表视图：展示所有需求卡片
- 卡片信息：标题、状态、关联任务数、产物数

### 8.2 任务看板页
- 四列看板：Backlog | In Progress | Review | Done
- 卡片：标题、进度条、实时日志预览
- 拖拽交互

### 8.3 产物云盘页
- 网格/列表视图切换
- 分类筛选
- 标签过滤
- 文件预览弹窗

---

## 9. 优先级建议

| 阶段 | 内容 | 优先级 |
|------|------|--------|
| MVP | 需求 + 任务看板 + Agent 集成 | P0 |
| V2 | 产物云盘 + 文件上传 | P1 |
| V3 | 多 Agent 并行 | P2 |
| V4 | 权限控制 | P3 |
