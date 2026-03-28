# Trace 统一实现参考文档

## 来源
从 `task-required-fields` 分支提取，基于 main 分支已实现的内容整理。

---

## 1. trace 包（backend/infrastructure/trace/context.go）

**main 和 task-required-fields 无差异，已在 main 分支。**

### 核心功能

```go
// 从 context 获取 trace 信息（不存在则生成新的）
traceID := trace.GetTraceID(ctx)
spanID := trace.MustGetSpanID(ctx)  // 不存在返回空字符串

// 开始新的 span，自动继承 traceID，设置 parentSpanID
subCtx, subSpanID := trace.StartSpan(ctx)

// 开始新的 trace
ctx, traceID, spanID := trace.StartTrace(ctx)

// 向 context 注入 trace 信息
ctx = trace.WithTraceID(ctx, traceID)
ctx = trace.WithSpanID(ctx, spanID)
ctx = trace.WithParentSpanID(ctx, parentSpanID)
```

### 关键函数

| 函数 | 说明 |
|------|------|
| `GetTraceID(ctx)` | 获取 traceID，不存在则生成新的 |
| `MustGetSpanID(ctx)` | 获取 spanID，不存在返回空字符串 |
| `GetParentSpanID(ctx)` | 获取 parentSpanID，不存在返回空字符串 |
| `StartSpan(ctx)` | 开始新 span，返回 (newCtx, newSpanID) |
| `StartTrace(ctx)` | 开始新 trace，返回 (newCtx, traceID, spanID) |

---

## 2. agent_handler.go（backend/application/agent_handler.go）

**main 和 task-required-fields 无功能差异。**

### CreateSubTasksFromLLM 子任务创建

```go
func CreateSubTasksFromLLM(ctx context.Context, task *domain.Task, repo domain.TaskRepository, plan *llm.SubTaskPlan) ([]string, error) {
    for _, st := range plan.SubTasks {
        // 使用 trace.StartSpan 从 context 自动获取新 spanID，parentSpanID 自动注入
        subCtx, subSpanID := trace.StartSpan(ctx)

        subTask, err := domain.NewTask(
            domain.NewTaskID(subTaskID),
            domain.NewTraceID(trace.GetTraceID(subCtx)),  // 从 subCtx 获取，保持 traceID 一致
            domain.NewSpanID(subSpanID),
            parentID,
            st.Goal,
            desc,
            taskType,
            taskRequirement,
            acceptanceCriteria,
            DefaultTaskTimeout,
            0,
            0,
        )

        subTask.SetDepth(getCurrentDepth(task))
        subTask.SetParentSpan(trace.GetParentSpanID(subCtx))  // 从 subCtx 获取 parentSpanID
    }
}
```

### finishAgentTask 任务完成

**main 分支（正确）：**
```go
func finishAgentTask(task *domain.Task, repo domain.TaskRepository) error {
    taskConclusion := task.TaskConclusion()
    if taskConclusion == "" {
        taskConclusion = "Agent 任务完成"
    }
    task.SetTaskConclusion(taskConclusion)
    task.Complete()  // 无参数
    return nil
}
```

**task-required-fields 分支（多余）：**
```go
// 多了 Result 相关代码，main 分支已移除
result := domain.NewResult(nil, taskConclusion)
task.Complete(result)
```

---

## 3. auto_task_executor.go（backend/application/auto_task_executor.go）

**main 和 task-required-fields 差异仅在 Result 相关代码。**

### ExecuteAutoTask 获取 trace 信息

```go
func (e *AutoTaskExecutor) ExecuteAutoTask(ctx context.Context, task *domain.Task) error {
    taskID := task.ID().String()
    // 从 context 获取 trace 信息，不再从 task 提取
    traceID := trace.GetTraceID(ctx)
    spanID := trace.MustGetSpanID(ctx)

    currentDepth := task.Depth() + 1
    // ...
}
```

### finishTask 任务完成

**main 分支（正确）：**
```go
func (e *AutoTaskExecutor) finishTask(task *domain.Task) error {
    taskConclusion := task.TaskConclusion()
    // ... 收集子任务结论 ...

    if taskConclusion == "" {
        taskConclusion = "任务完成"
    }
    task.SetTaskConclusion(taskConclusion)
    task.Complete()  // 无参数
    return nil
}
```

**task-required-fields 分支（多余）：**
```go
// 多了 Result 相关代码
result := domain.NewResult(nil, taskConclusion)
task.Complete(result)
```

### updateParentWithChildResult

**main 分支：** 直接 `e.repo.Save(context.Background(), parent)`

**task-required-fields 分支：** 多了 Result 更新逻辑（main 已移除）

---

## 4. create_task_tool.go（backend/infrastructure/llm/tools/task/create_task_tool.go）

**main 和 task-required-fields 无功能差异。**

### 从 ctx 提取 trace 信息

```go
func (t *CreateTaskTool) Execute(ctx context.Context, input json.RawMessage) (*llm.ToolResult, error) {
    // 设置 TraceID 和 SpanID（从 ctx 提取）
    traceIDStr := trace.GetTraceID(ctx)
    spanIDStr := trace.MustGetSpanID(ctx)
    if traceIDStr != "" {
        cmd.TraceID = &traceIDStr
    }
    if spanIDStr != "" {
        cmd.SpanID = &spanIDStr
    }

    // 设置 ParentSpanID（从 ctx 提取）
    parentSpanID := trace.GetParentSpanID(ctx)
    if parentSpanID == "" {
        // 尝试从 HookContext metadata 获取
        if hc, ok := ctx.(*domain.HookContext); ok {
            parentSpanID = hc.GetMetadata("span_id")
        }
    }
    if parentSpanID != "" {
        cmd.ParentSpanID = parentSpanID
    }
}
```

---

## 5. processor.go（backend/pkg/channel/processor.go）toolHookAdapter

**PreToolCall 中设置 trace context：**

```go
func (a *toolHookAdapter) PreToolCall(toolName string, input json.RawMessage) (json.RawMessage, error) {
    // ... 构建 callCtx ...

    // 将 tool_call 的 span_id 设置到 hookCtx 的 metadata 中
    a.hookCtx.SetMetadata("span_id", a.spanID)
    a.hookCtx.SetMetadata("parent_span_id", a.parentSpanID)

    // 使用 trace.WithSpanID 设置 span，供工具执行时通过 trace.GetSpanID 获取
    execCtx := trace.WithSpanID(ctxWithScope, a.spanID)
    if a.parentSpanID != "" {
        execCtx = trace.WithParentSpanID(execCtx, a.parentSpanID)
    }
    a.currentCtx = execCtx

    return input, nil
}
```

---

## 6. 结论

**task-required-fields 分支相比 main 的 trace 实现：**
- ✅ trace context 使用方式一致
- ✅ StartSpan/StartTrace 使用方式一致
- ❌ 多了 `Result` 值对象使用（main 已移除）
- ❌ 多了 `UpdateResult` 方法（main 已移除）
- ❌ `Complete()` 需要传参（main 改为无参）

**建议：基于 main 分支实现，无需从 task-required-fields 合并。trace 相关功能已在 main 中完整实现。**
