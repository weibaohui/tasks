# AI DevOps 数据模型设计

## 一、设计思想

**核心原则：扁平结构，外键关联**

- 所有嵌套结构都拆成独立表
- 通过外键建立关系
- 不使用 JSON 数组字段

## 二、数据表定义

### 2.1 task_records（任务记录表）

```sql
CREATE TABLE task_records (
    id TEXT PRIMARY KEY,

    -- 基本信息
    title TEXT NOT NULL,
    description TEXT,

    -- 溯源关系
    source_type TEXT NOT NULL,  -- requirement, task, manual
    source_id TEXT NOT NULL,

    -- 当前状态
    stage TEXT NOT NULL DEFAULT 'created',
    status TEXT NOT NULL DEFAULT 'pending',
    percent INTEGER DEFAULT 0,

    -- 验收标准
    acceptance_criteria TEXT,

    -- 归属
    assignee TEXT,
    confirmed_by TEXT,
    confirmed_at INTEGER,

    -- 时间戳
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
```

### 2.2 stage_transitions（阶段流转记录表）

```sql
CREATE TABLE stage_transitions (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    from_stage TEXT NOT NULL,
    to_stage TEXT NOT NULL,
    transition_by TEXT NOT NULL,
    transition_at INTEGER NOT NULL,
    reason TEXT,
    note TEXT,
    FOREIGN KEY (task_id) REFERENCES task_records(id)
);
```

### 2.3 evidence_records（证据记录表）

```sql
CREATE TABLE evidence_records (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    type TEXT NOT NULL,  -- code, doc, test_result, screenshot
    path TEXT NOT NULL,
    description TEXT,
    created_by TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (task_id) REFERENCES task_records(id)
);
```

### 2.4 comments（评论表）

```sql
CREATE TABLE comments (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    author TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (task_id) REFERENCES task_records(id)
);
```

### 2.5 task_relations（任务关系表）

记录任务之间的依赖关系。

```sql
CREATE TABLE task_relations (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    depends_on_id TEXT NOT NULL,
    relation_type TEXT NOT NULL,  -- blocks, related_to, spawned_from
    created_at INTEGER NOT NULL,
    FOREIGN KEY (task_id) REFERENCES task_records(id),
    FOREIGN KEY (depends_on_id) REFERENCES task_records(id)
);
```

### 2.6 stage_definitions（阶段定义表）

定义系统的所有阶段。

```sql
CREATE TABLE stage_definitions (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,  -- created, analyzing, designing, developed, testing, verified, deployed, accepted
    name TEXT NOT NULL,         -- 创建, 分析中, 设计中, 开发完成, 测试中, 测试通过, 已部署, 已验收
    sequence INTEGER NOT NULL,   -- 顺序
    is_auto_transition INTEGER DEFAULT 0,  -- 是否自动流转
    is_gate INTEGER DEFAULT 0    -- 是否需要人工确认
);
```

## 三、Go 结构体定义

```go
// TaskRecord 任务记录
type TaskRecord struct {
    ID          string `json:"id"`
    Title       string `json:"title"`
    Description string `json:"description"`

    // 溯源
    SourceType string `json:"source_type"`
    SourceID   string `json:"source_id"`

    // 状态
    Stage   string `json:"stage"`
    Status  string `json:"status"`
    Percent int    `json:"percent"`

    // 验收
    AcceptanceCriteria string `json:"acceptance_criteria"`

    // 归属
    Assignee    string `json:"assignee"`
    ConfirmedBy string `json:"confirmed_by"`
    ConfirmedAt int64  `json:"confirmed_at"`

    // 时间
    CreatedAt int64 `json:"created_at"`
    UpdatedAt int64 `json:"updated_at"`
}

// StageTransition 阶段流转
type StageTransition struct {
    ID            string `json:"id"`
    TaskID        string `json:"task_id"`
    FromStage     string `json:"from_stage"`
    ToStage       string `json:"to_stage"`
    TransitionBy  string `json:"transition_by"`
    TransitionAt  int64  `json:"transition_at"`
    Reason        string `json:"reason"`
    Note          string `json:"note"`
}

// EvidenceRecord 证据记录
type EvidenceRecord struct {
    ID          string `json:"id"`
    TaskID      string `json:"task_id"`
    Type        string `json:"type"`
    Path        string `json:"path"`
    Description string `json:"description"`
    CreatedBy   string `json:"created_by"`
    CreatedAt   int64  `json:"created_at"`
}

// TaskRelation 任务关系
type TaskRelation struct {
    ID          string `json:"id"`
    TaskID      string `json:"task_id"`
    DependsOnID string `json:"depends_on_id"`
    RelationType string `json:"relation_type"`
    CreatedAt   int64  `json:"created_at"`
}

// Comment 评论
type Comment struct {
    ID        string `json:"id"`
    TaskID    string `json:"task_id"`
    Author    string `json:"author"`
    Content   string `json:"content"`
    CreatedAt int64  `json:"created_at"`
}
```

## 四、表关系图

```
┌─────────────────┐
│  task_records   │  1 ──────▶ *  stage_transitions
│                 │  1 ──────▶ *  evidence_records
│  - id (PK)      │  1 ──────▶ *  comments
│  - title        │  1 ──────▶ *  task_relations
│  - source_type  │
│  - source_id    │         *  ──────▶ 1  task_relations
│  - stage        │                        (反向依赖)
│  - status       │
└─────────────────┘

┌─────────────────┐
│ stage_definitions │  1 ──────▶ *  task_records
│                    │           (通过 stage 字段关联)
│  - code          │
│  - sequence      │
└─────────────────┘
```

## 五、阶段定义数据

```sql
INSERT INTO stage_definitions (id, code, name, sequence, is_auto_transition, is_gate) VALUES
('stage_1', 'created', '创建', 1, 1, 0),
('stage_2', 'analyzing', '分析中', 2, 0, 0),
('stage_3', 'designing', '设计中', 3, 0, 0),
('stage_4', 'developed', '开发完成', 4, 1, 0),
('stage_5', 'testing', '测试中', 5, 0, 0),
('stage_6', 'verified', '测试通过', 6, 1, 0),
('stage_7', 'deployed', '已部署', 7, 1, 0),
('stage_8', 'accepted', '已验收', 8, 0, 1);
```

## 六、查询示例

### 6.1 查询需求下的所有任务

```sql
SELECT * FROM task_records
WHERE source_type = 'requirement' AND source_id = 'req_xxx'
ORDER BY created_at;
```

### 6.2 查询任务的依赖任务

```sql
SELECT tr.* FROM task_records tr
INNER JOIN task_relations trel ON tr.id = trel.depends_on_id
WHERE trel.task_id = 'task_xxx';
```

### 6.3 查询任务的证据材料

```sql
SELECT * FROM evidence_records
WHERE task_id = 'task_xxx'
ORDER BY created_at;
```

### 6.4 查询任务流转历史

```sql
SELECT * FROM stage_transitions
WHERE task_id = 'task_xxx'
ORDER BY transition_at;
```

### 6.5 查询待确认的任务（Gate 点）

```sql
SELECT * FROM task_records
WHERE stage IN ('analyzing', 'designing', 'accepted')
AND confirmed_at IS NULL;
```

### 6.6 查询某 Agent 的待办任务

```sql
SELECT * FROM task_records
WHERE assignee = 'agent_xxx'
AND status NOT IN ('completed', 'failed');
```

### 6.7 查询可执行的任务（依赖已全部完成）

```sql
SELECT tr.* FROM task_records tr
WHERE tr.status = 'pending'
AND NOT EXISTS (
    SELECT 1 FROM task_relations trel
    INNER JOIN task_records dep ON trel.depends_on_id = dep.id
    WHERE trel.task_id = tr.id
    AND dep.status != 'completed'
);
```
