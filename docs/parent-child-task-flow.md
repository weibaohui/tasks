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
TaskStatusRunning   (1)  // 执行中
    ↓
TaskStatusPendingSummary (2)  // 等待总结（所有子任务完成，等待 TaskSummarizer 生成总结）
    ↓
TaskStatusCompleted (3)  // 已完成
```

**状态转换规则：**
- `Pending → Running`：任务开始执行
- `Running → PendingSummary`：父任务所有子任务完成，等待生成总结
- `PendingSummary → Completed`：TaskSummarizer 生成总结后完成任务

---

## 二、模型字段对照

### 2.1 Task 模型字段（domain/task.go）

| 模型字段 | 数据库字段 | 类型 | 说明 |
|---------|-----------|------|------|
| `Name` | `name` | string | 任务名称 |
| `TaskRequirement` | `task_requirement` | string | 任务要求 |
| `AcceptanceCriteria` | `acceptance_criteria` | string | 验收标准 |
| `TaskConclusion` | `task_conclusion` | string | **任务自身结论（必填）** |
| `SubtaskRecords` | `subtask_records` | string | **子任务成对文档汇总（仅父任务使用）** |
| `ParentID` | `parent_id` | *TaskID | 父任务ID（nil=根任务） |

### 2.2 子任务完成时写入的成对文档

```go
type TaskResultPair struct {
    TaskName            string     // 任务名称
    TaskRequirement     string     // 任务要求（从父任务继承）
    AcceptanceCriteria  string     // 验收标准（从父任务继承）
    TaskConclusion      string     // 任务结论（执行结果）
    CompletedAt         *time.Time `yaml:"completed_at,omitempty"`
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

## 三、TaskSummarizer 总结者

### 3.1 概述

`TaskSummarizer` 是专门负责生成父任务总结的服务，订阅 `TaskPendingSummary` 事件，异步处理总结生成。

### 3.2 工作流程

```
finishTask() 执行:
    ├── 子任务完成 → updateParentWithChildResult 写入 subtask_records
    ├── 父任务 finishTask → 设置 PendingSummary 状态 → 发布 TaskPendingSummary 事件 → 返回
    │
TaskSummarizer 订阅事件后:
    ├── 重新加载任务确保最新状态
    ├── 解析 subtask_records
    ├── 获取 LLM provider（复用对话上下文）
    ├── 构建总结 prompt
    ├── 调用 LLM 生成总结
    ├── 写入 task_conclusion
    └── 调用 task.Complete() → 进入 Completed 状态
```

### 3.3 LLM 调用上下文复用

TaskSummarizer 通过 `AutoTaskExecutor.llmLookup` 获取 LLM provider，自动继承：
- `traceID` / `spanID` / `parentSpanID` - 追踪链路
- `agentCode` / `userCode` / `channelCode` - 上下文标识
- `sessionKey` - 会话信息
- Hook 配置 - LLM 调用钩子

### 3.4 核心代码

```go
// TaskSummarizer 处理 PendingSummary 事件
func (s *TaskSummarizer) HandlePendingSummary(event domain.DomainEvent) {
    pendingEvent := event.(*domain.TaskPendingSummaryEvent)
    task := pendingEvent.Task()

    // 重新加载任务
    task, _ = s.repo.FindByID(ctx, task.ID())

    // 解析 subtask_records
    pairs, _ := domain.ParseTaskResultPairs(task.SubtaskRecords())

    // 获取 LLM provider（复用上下文）
    provider, _ := s.executor.llmLookup.getProviderForTask(ctx, task)

    // 调用 LLM 生成总结
    summary, _ := s.generateSummary(ctx, task, pairs, provider)

    // 完成总结
    task.SetTaskConclusion(summary)
    task.Complete()
    s.repo.Save(ctx, task)
}
```

---

## 四、流程详细说明

### 4.1 子任务完成 → 写入 subtask_records

当子任务完成时，`finishTask` 调用 `updateParentWithChildResult`：

```go
func (e *AutoTaskExecutor) updateParentWithChildResult(task *domain.Task) {
    // 构建子任务成对文档
    pair := domain.TaskResultPair{
        TaskName:           task.Name(),
        TaskRequirement:    task.TaskRequirement(),
        AcceptanceCriteria: task.AcceptanceCriteria(),
        TaskConclusion:     task.TaskConclusion(),
        CompletedAt:        time.Now(),
        Status:             task.Status(),
    }

    // 追加到父任务的 subtask_records
    existingRecords := parent.SubtaskRecords()
    newRecords, _ := domain.AppendTaskResultPair(existingRecords, pair)
    parent.SetSubtaskRecords(newRecords)
    e.repo.Save(ctx, parent)
}
```

### 4.2 父任务完成 → PendingSummary

当父任务的 `finishTask` 执行时：

```go
func (e *AutoTaskExecutor) finishTask(task *domain.Task) error {
    // 收集子任务成对文档到 subtask_records
    // ...

    isParentWithSubtasks := task.ParentID() == nil && todoListStr != ""

    if isParentWithSubtasks {
        // 保存 subtask_records
        e.saveTaskPreservingMetadata(task)

        // 进入 PendingSummary 状态
        task.PendingSummary()
        e.saveTaskPreservingMetadata(task)

        // 发布事件，TaskSummarizer 异步处理
        evt := domain.NewTaskPendingSummaryEvent(task)
        e.eventBus.Publish(evt)

        return nil  // 立即返回，不等待总结完成
    }

    // 非父任务直接完成
    task.SetTaskConclusion("任务完成")
    task.Complete()
    // ...
}
```

### 4.3 TaskSummarizer 生成总结

```go
func (s *TaskSummarizer) generateSummary(ctx context.Context, task *domain.Task, pairs []domain.TaskResultPair, provider llm.LLMProvider) (string, error) {
    var sb strings.Builder

    sb.WriteString("## 任务总结\n\n")
    sb.WriteString("### 任务要求\n")
    sb.WriteString(task.TaskRequirement())
    sb.WriteString("\n\n")

    if task.AcceptanceCriteria() != "" {
        sb.WriteString("### 验收标准\n")
        sb.WriteString(task.AcceptanceCriteria())
        sb.WriteString("\n\n")
    }

    sb.WriteString("### 子任务完成情况\n")
    for i, pair := range pairs {
        sb.WriteString(fmt.Sprintf("#### %d. %s\n", i+1, pair.TaskName))
        sb.WriteString(fmt.Sprintf("- 要求: %s\n", pair.TaskRequirement))
        sb.WriteString(fmt.Sprintf("- 结论: %s\n", pair.TaskConclusion))
        sb.WriteString(fmt.Sprintf("- 状态: %s\n", pair.Status))
    }

    sb.WriteString("\n### 综合分析\n请根据以上子任务完成情况，生成综合总结。")

    // 调用 LLM
    return provider.Generate(ctx, sb.String())
}
```

---

## 五、状态转换图

```
┌─────────────────────────────────────────────────────────────────────┐
│                           父任务                                      │
│  ┌─────────┐    ┌─────────┐    ┌────────────────┐    ┌─────────┐│
│  │ Pending │───▶│ Running │───▶│PendingSummary  │───▶│Completed││
│  └─────────┘    └─────────┘    └────────────────┘    └─────────┘│
│      │              │                │                  ▲         │
│      │              │                │                  │         │
│      │              │                ▼                  │         │
│      │              │         ┌────────────┐           │         │
│      │              │         │TaskSummarizer│          │         │
│      │              │         │  (异步)     │──────────┘         │
│      │              │         └────────────┘                    │
│      │              │                │                           │
└──────│──────────────│────────────────│───────────────────────────┘
       │              │                │
       ▼              ▼                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                           子任务                                      │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐                       │
│  │ Pending │───▶│ Running │───▶│Completed│                       │
│  └─────────┘    └─────────┘    └─────────┘                       │
│       │              │              │                              │
│       │              │              ▼                              │
│       │              │    updateParentWithChildResult()            │
│       │              │    (写入父任务 subtask_records)              │
└──────│──────────────│─────────────────────────────────────────────┘
```

---

## 六、数据模型

### 6.1 Task 实体字段（domain/task.go）

```go
type Task struct {
    ID                   TaskID
    ParentID            *TaskID      // nil 表示根任务
    Status              TaskStatus
    Name                string       // 任务名称
    TaskRequirement     string       // 任务要求
    AcceptanceCriteria  string       // 验收标准
    TaskConclusion      string       // 任务结论（必填，自身结果）
    SubtaskRecords     string       // YAML: 子任务成对文档汇总
    // ...
}
```

### 6.2 字段职责说明

| 字段 | 父任务 | 子任务 | 说明 |
|------|-------|-------|------|
| `name` | 任务名称 | 继承或覆盖 | 任务标识 |
| `task_requirement` | 任务要求 | 继承父任务 | 任务描述 |
| `acceptance_criteria` | 验收标准 | 继承父任务 | 完成标准 |
| `task_conclusion` | **TaskSummarizer 生成的总结** | 执行结论 | **任务自身结果** |
| `subtask_records` | 子任务成对文档汇总 | 不适用 | 仅父任务使用，YAML格式 |

---

## 七、关键约束

1. **子任务必须成对记录**：`task_name + task_requirement + acceptance_criteria + task_conclusion` 必须完整
2. **SubtaskRecords 保留**：汇总后保留所有子任务的原始成对文档
3. **TaskConclusion 由 TaskSummarizer 生成**：父任务的 `task_conclusion` 是 LLM 总结后的文档
4. **PendingSummary 是中间状态**：`finishTask` 设置后立即返回，不阻塞
5. **总结异步执行**：TaskSummarizer 通过事件驱动，不影响子任务执行流程

---

## 八、流程总结

```
1. [父任务] Pending → Running
   └── 生成子任务列表（继承 task_requirement + acceptance_criteria）

2. [子任务] Running → Completed
   └── 子任务写入 task_conclusion
   └── updateParentWithChildResult() 写入父任务 subtask_records

3. [父任务] 所有子任务完成 → PendingSummary
   └── finishTask() 设置 PendingSummary 状态
   └── 发布 TaskPendingSummary 事件
   └── 立即返回

4. [TaskSummarizer] 订阅事件，异步处理
   └── 重新加载任务
   └── 解析 subtask_records
   └── 调用 LLM 生成总结（复用上下文）
   └── 写入 task_conclusion
   └── 调用 Complete() → Completed
```

---

## 九、YAML 格式说明

### 9.1 SubtaskRecords 字段格式

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

### 9.2 解析方式

```go
// 解析 SubtaskRecords 字段
func ParseTaskResultPairs(records string) ([]TaskResultPair, error) {
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

## 十、实现要点

### 10.1 新增组件

| 组件 | 文件 | 说明 |
|------|------|------|
| `TaskSummarizer` | `application/task_summarizer.go` | 总结者服务 |
| `TaskPendingSummaryEvent.Task()` | `domain/event.go` | 导出方法 |
| `Task.PendingSummary()` | `domain/task.go` | 进入等待总结状态 |
| `TaskResultPair` | `domain/task.go` | 成对文档结构体 |

### 10.2 数据库迁移

```sql
ALTER TABLE tasks ADD COLUMN subtask_records TEXT;
```

### 10.3 初始化

```go
// main.go
autoExecutor := application.NewAutoTaskExecutor(...)
autoExecutor.SetRepositories(agentRepo, providerRepo, channelRepo, llmFactory)

// 初始化并启动 TaskSummarizer
summarizer := application.NewTaskSummarizer(taskRepo, autoExecutor, eventBus)
summarizer.Start()
```

---

## 十一、完整文档树示例

```
根任务
├── task_requirement: "分析产品运营数据"
├── acceptance_criteria: "包含用户增长、留存、转化三个维度"
├── task_conclusion: "综合分析报告（TaskSummarizer 生成）"
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
