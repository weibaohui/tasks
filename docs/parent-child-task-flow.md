# 父子任务完成流程设计

## 一、核心概念

### 1.1 作业模型（类比学校发作业/收作业）

| 学校场景 | 任务系统对应 | 模型字段 | 说明 |
|---------|-------------|---------|------|
| 老师发作业 | 父任务 | `name` + `taskRequirement` + `acceptanceCriteria` | 布置任务要求 |
| 学生做作业 | 子任务 | 继承父任务 | 每人独立执行 |
| 学生交作业 | 子任务结论 | 写入父任务 `subtaskRecords` | 任务名称 + 要求 + 验收标准 + 结论 |
| 老师收作业 | 汇总子任务结果 | 从 `subtaskRecords` 收集 | 收集所有成对文档 |
| 老师总结 | 父任务结论 | `taskConclusion` | 结合任务要求 + 验收标准 + 作业结果 |

### 1.2 任务状态机

```
TaskStatusPending   (0)  // 待处理
    ↓
TaskStatusDecomposed (1)  // 已分解（生成子任务）
    ↓
TaskStatusRunning   (2)  // 执行中
    ↓
TaskStatusCompleted (3)  // 已完成
```

**状态转换规则：**
- `Pending → Decomposed`：父任务完成任务分解，生成子任务
- `Decomposed → Running`：所有子任务开始执行
- `Running → Completed`：所有子任务完成 + 父任务总结完成

---

## 二、模型字段对照

### 2.1 Task 模型字段（domain/task.go）

| 模型字段 | 数据库字段 | 类型 | 说明 |
|---------|-----------|------|------|
| `Name` | `name` | string | 任务名称 |
| `TaskRequirement` | `task_requirement` | string | 任务要求 |
| `AcceptanceCriteria` | `acceptance_criteria` | string | 验收标准 |
| `TaskConclusion` | `task_conclusion` | string | **任务自身结论（必填）** |
| `SubtaskRecords` | `subtask_records` | string | **子任务成对文档汇总（仅父任务使用，需新增）** |
| `ParentID` | `parent_id` | *TaskID | 父任务ID（nil=根任务） |

### 2.2 子任务完成时写入的成对文档

```go
type TaskResultPair struct {
    TaskName            string    // 任务名称
    TaskRequirement     string    // 任务要求（从父任务继承）
    AcceptanceCriteria  string    // 验收标准（从父任务继承）
    TaskConclusion      string    // 任务结论（执行结果）
    CompletedAt         time.Time
    Status              TaskStatus
}
```

**存储位置：**
- 每个子任务完成后，将其 `TaskResultPair` 写入父任务的 `SubtaskRecords` 字段
- `SubtaskRecords` 格式：YAML，每个子任务占一个文档块，之间用 `---` 分隔

```yaml
# === 子任务 1 ===
task_name: "数据分析-用户增长"
task_requirement: "分析用户增长数据，识别关键指标"
acceptance_criteria: "输出包含DAU、MAU、新增用户数、同比环比数据"
task_conclusion: "发现DAU环比增长5%，主要由新功能带动，新用户次日留存率提升3%"
completed_at: "2026-03-28T10:00:00Z"
status: "completed"
---
# === 子任务 2 ===
task_name: "数据分析-用户留存"
task_requirement: "分析用户留存情况"
acceptance_criteria: "输出包含次日留存、周留存、月留存数据，对比行业基准"
task_conclusion: "周留存率为35%，低于目标40%，需优化新手引导流程"
completed_at: "2026-03-28T10:05:00Z"
status: "completed"
```

---

## 三、父任务状态响应

### 3.1 触发时机

当任意子任务状态变更时（Completed/Failed/Cancelled），父任务应收到通知并更新状态。

### 3.2 父任务状态检查逻辑

```go
// AutoTaskExecutor.checkSubTasksStatus()
func (e *AutoTaskExecutor) checkSubTasksStatus(ctx context.Context, parentID string) error {
    subTasks, err := e.repo.FindByParentID(ctx, parentID)
    if err != nil {
        return err
    }

    var completedCount, failedCount, totalCount int
    for _, subTask := range subTasks {
        totalCount++
        switch subTask.Status() {
        case TaskStatusCompleted:
            completedCount++
        case TaskStatusFailed:
            failedCount++
        }
    }

    // 更新父任务进度
    progress := float64(completedCount) / float64(totalCount) * 100

    // 检查是否所有子任务完成
    if completedCount + failedCount == totalCount {
        if failedCount == 0 {
            // 所有子任务成功，等待父任务总结
            return e.transitionParentToPendingSummary(parentID)
        } else {
            // 部分失败，需要决策如何处理
            return e.handlePartialFailure(parentID, failedCount, totalCount)
        }
    }

    return nil
}
```

### 3.3 父任务状态转换

| 子任务状态 | 父任务状态 | 说明 |
|-----------|-----------|------|
| 部分完成 | Running | 等待其他子任务 |
| 全部完成 | PendingSummary | 所有子任务完成，等待撰写总结 |
| 全部失败 | Failed | 子任务全部失败 |

---

## 四、汇总与总结流程

### 4.1 汇总阶段（Collecting）

当所有子任务完成后，进入汇总阶段：

1. **收集成对文档**：从每个子任务收集 `task_name + task_requirement + acceptance_criteria + task_conclusion`
2. **写入父任务 SubtaskRecords**：将所有成对文档以 YAML 格式写入父任务的 `SubtaskRecords` 字段
3. **准备总结输入**：汇总所有成对文档 + 父任务原始要求 + 验收标准

### 4.2 总结阶段（Summarizing）

```go
// 总结阶段输入
type SummaryInput struct {
    ParentRequirement      string          // 父任务 task_requirement
    ParentAcceptanceCriteria string        // 父任务 acceptance_criteria
    SubtaskRecords        []TaskResultPair // 子任务成对文档列表
}

// 总结阶段输出
type SummaryOutput struct {
    ParentConclusion  string         // 父任务结论（写入 task_conclusion）
    Summary           string         // 综合总结
    ChildConclusions  []string       // 保留原始子任务结论引用
}
```

### 4.3 完成闭环

```
┌─────────────────────────────────────────────────────────────┐
│                        父任务                                │
│  ┌─────────┐    ┌─────────┐    ┌──────────┐    ┌─────────┐  │
│  │ Pending │───▶│Decomposed│───▶│ Running  │───▶│Completed│  │
│  └─────────┘    └─────────┘    └──────────┘    └─────────┘  │
│      │              │              │              │         │
│      │         生成子任务        检查状态        总结        │
│      │              │              │              │         │
└──────│──────────────│──────────────│──────────────│─────────┘
       │              │              │              │
       ▼              ▼              ▼              ▼
┌─────────────────────────────────────────────────────────────┐
│                        子任务                                │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  子任务完成后，贡献成对文档到父任务的 subtask_records │    │
│  │  subtask_records (YAML):                            │    │
│  │  - task_name: ...                                  │    │
│  │    task_requirement: ...                           │    │
│  │    acceptance_criteria: ...                        │    │
│  │    task_conclusion: ...                            │    │
│  │  - task_name: ...                                  │    │
│  │    task_requirement: ...                           │    │
│  │    acceptance_criteria: ...                        │    │
│  │    task_conclusion: ...                            │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  父任务 subtask_records: 所有子任务成对文档汇总（保留）      │
│  父任务 task_conclusion: 总结文档（最终输出）                │
└─────────────────────────────────────────────────────────────┘
```

---

## 五、数据模型

### 5.1 Task 实体字段（domain/task.go）

```go
type Task struct {
    ID                   TaskID
    ParentID            *TaskID      // nil 表示根任务
    Status              TaskStatus
    Name                string       // 任务名称
    TaskRequirement     string       // 任务要求
    AcceptanceCriteria  string       // 验收标准
    TaskConclusion      string       // 任务结论（必填，自身结果）
    SubtaskRecords      string       // YAML: 子任务成对文档汇总（仅父任务使用，需新增）
    // ...
}
```

### 5.2 字段职责说明

| 字段 | 父任务 | 子任务 | 说明 |
|------|-------|-------|------|
| `name` | 任务名称 | 继承或覆盖 | 任务标识 |
| `task_requirement` | 任务要求 | 继承父任务 | 任务描述 |
| `acceptance_criteria` | 验收标准 | 继承父任务 | 完成标准 |
| `task_conclusion` | 最终总结文档 | 执行结论 | **任务自身结果** |
| `subtask_records` | 子任务成对文档汇总 | 不适用 | 仅父任务使用，YAML格式 |

---

## 六、关键约束

1. **子任务必须成对记录**：`task_name + task_requirement + acceptance_criteria + task_conclusion` 必须完整
2. **SubtaskRecords 保留**：汇总后保留所有子任务的原始成对文档
3. **TaskConclusion 是最终输出**：父任务的 `task_conclusion` 是总结后的文档，不是子任务结论的拼接
4. **状态转换驱动**：
   - 子任务完成 → 触发父任务检查
   - 所有子任务完成 → 父任务进入 `PendingSummary` 状态
   - 父任务总结完成 → 父任务进入 `Completed` 状态

---

## 七、流程总结

```
1. [父任务] Pending → Decomposed
   └── 生成子任务列表（继承 task_requirement + acceptance_criteria）

2. [子任务] Running → Completed
   └── 子任务写入 task_conclusion
   └── 父任务收集子任务成对文档到 subtask_records

3. [父任务] 检测到子任务完成
   └── 更新进度，检查是否全部完成

4. [父任务] 所有子任务完成 → PendingSummary
   └── subtask_records 已包含所有子任务成对文档
   └── 结合父任务 task_requirement + acceptance_criteria 生成总结
   └── 写入父任务的 task_conclusion

5. [父任务] 总结完成 → Completed
   └── task_conclusion = 最终总结文档
   └── subtask_records = 原始子任务成对文档（保留）
   └── 整个任务树完成
```

---

## 八、YAML 格式说明

### 8.1 SubtaskRecords 字段格式

多个子任务时，使用 `---` 分隔：

```yaml
# === 子任务 1 ===
task_name: "子任务名称"
task_requirement: "继承的任务要求"
acceptance_criteria: "继承的验收标准"
task_conclusion: "子任务执行结论"
completed_at: "2026-03-28T10:00:00Z"
status: "completed"
---
# === 子任务 2 ===
task_name: "子任务名称"
task_requirement: "继承的任务要求"
acceptance_criteria: "继承的验收标准"
task_conclusion: "子任务执行结论"
completed_at: "2026-03-28T10:05:00Z"
status: "completed"
```

### 8.2 解析方式

```go
// 解析 SubtaskRecords 字段
func parseSubtaskRecords(records string) ([]TaskResultPair, error) {
    // 按 --- 分隔 YAML 文档
    docs := strings.Split(records, "---")
    var pairs []TaskResultPair
    for _, doc := range docs {
        doc = strings.TrimSpace(doc)
        if doc == "" || strings.HasPrefix(doc, "#") {
            continue
        }
        var pair TaskResultPair
        if err := yaml.Unmarshal([]byte(doc), &pair); err != nil {
            return nil, err
        }
        pairs = append(pairs, pair)
    }
    return pairs, nil
}
```

---

## 九、实现要点

### 9.1 需要新增的字段

| 字段 | 类型 | 位置 | 说明 |
|------|------|------|------|
| `subtask_records` | string | Task 模型 + 数据库 | 存储子任务成对文档汇总 |

### 9.2 数据库迁移

```sql
ALTER TABLE tasks ADD COLUMN subtask_records TEXT;
```

### 9.3 Task 模型变更

```go
// domain/task.go
type Task struct {
    // ... 现有字段 ...
    SubtaskRecords  string  // YAML: 子任务成对文档汇总（仅父任务使用）
}
```

---

## 十、完整文档树示例

```
根任务
├── task_requirement: "分析产品运营数据"
├── acceptance_criteria: "包含用户增长、留存、转化三个维度"
├── task_conclusion: "综合分析报告（总结文档）"
├── subtask_records:
│   ├── # === 子任务 1 ===
│   │   task_name: "用户增长分析"
│   │   task_requirement: "分析用户增长数据"
│   │   acceptance_criteria: "包含DAU、MAU、新增..."
│   │   task_conclusion: "DAU环比增长5%..."
│   │
│   └── # === 子任务 2 ===
│       task_name: "留存分析"
│       task_requirement: "分析留存情况"
│       acceptance_criteria: "包含次日、周、月留存..."
│       task_conclusion: "周留存35%，低于40%目标..."
│
└── [子任务节点...]
```
