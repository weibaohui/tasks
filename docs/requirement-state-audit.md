# 需求状态修改位置审计报告

## 概述

本文档记录项目中所有需求（Requirement）状态字段的修改位置，以便评估是否需要重构，确保状态更新源唯一（通过 CLI 组合命令 `taskmanager statemachine execute` + `taskmanager requirement update-state`）。

---

## 状态定义

```go
// RequirementStatus 需求状态
type RequirementStatus string

const (
    RequirementStatusTodo       RequirementStatus = "todo"        // 待处理
    RequirementStatusPreparing  RequirementStatus = "preparing"   // 准备中
    RequirementStatusCoding     RequirementStatus = "coding"      // 编码中
    RequirementStatusPROpened   RequirementStatus = "pr_opened"   // PR已创建
    RequirementStatusFailed     RequirementStatus = "failed"      // 失败
    RequirementStatusCompleted  RequirementStatus = "completed"   // 已完成
    RequirementStatusDone       RequirementStatus = "done"        // 已结束
)
```

---

## 一、领域层状态修改方法 (domain/requirement.go)

| 方法 | 状态变更 | 调用场景 | 是否应保留 |
|------|---------|---------|-----------|
| `StartDispatch()` | todo → preparing | 开始派发需求 | ❌ 应由 AI/Hook 触发 |
| `MarkCoding()` | preparing → coding | 分身创建成功 | ❌ 应由 AI/Hook 触发 |
| `MarkPROpened()` | coding → pr_opened | PR 被创建 | ❌ 应由 AI/Hook 触发 |
| `MarkFailed()` | any → failed | 执行失败 | ❌ 应由 AI/Hook 触发 |
| `MarkCompleted()` | any → completed | 正常完成 | ❌ 应由 AI/Hook 触发 |
| `Redispatch()` | any → todo | 重新派发 | ⚠️ 特殊，人工操作 |

---

## 二、应用层调用位置

### 2.1 requirement_dispatch_service.go (派发服务)

```go
// Line 105: 派发开始时设置 preparing
if err := requirement.StartDispatch(cmd.AgentCode); err != nil {
    return nil, err
}

// Line 118: 分身创建失败时设置 failed
requirement.MarkFailed(err.Error())

// Line 123: 分身创建成功时设置 coding
if err := requirement.MarkCoding(workspacePath, replicaAgent.AgentCode().String()); err != nil {
    return nil, err
}

// Line 131, 137, 143: 各种错误情况设置 failed
requirement.MarkFailed(err.Error())
```

**分析：** 这些状态变更都是派发流程中的自动步骤，**硬编码了状态流转逻辑**，与状态机设计冲突。

### 2.2 requirement_service.go (需求服务)

```go
// Line 154: 报告 PR 创建时设置 pr_opened
func (s *RequirementApplicationService) ReportRequirementPROpened(...) {
    requirement.MarkPROpened()
}

// Line 171: 重新派发时重置为 todo
func (s *RequirementApplicationService) RedispatchRequirement(...) {
    if err := requirement.Redispatch(); err != nil {
        return nil, err
    }
}
```

**分析：**
- `MarkPROpened()` - 由 PR webhook 触发，**应该改为调用状态机执行转换**
- `Redispatch()` - 人工重新派发操作，属于特殊流程，可以保留

### 2.3 heartbeat_scheduler.go (心跳调度器)

```go
// Line 143: 清理过期需求时设置 failed
req.MarkFailed("cleanup: " + reason)
```

**分析：** 清理过期需求时标记失败，这是系统维护操作，**可以保留**。

---

## 三、基础设施层调用位置

### 3.1 claudecode/processor.go (Claude Code 处理器)

```go
// Line 1101: Claude Code 成功完成时设置 completed
if success {
    requirement.MarkCompleted()
    requirement.SetClaudeRuntimeResult(finalResult)
    if err := p.requirementRepo.Save(ctx, requirement); err != nil {
        // ...
    }
}
```

**分析：** 这是**最大的冲突点**！Claude Code 完成后自动标记需求为 completed，完全绕过状态机。这与我们的设计目标冲突——AI 应该根据工作结果决定是否推进状态，而不是系统自动完成。

---

## 四、前端调用位置

### 4.1 HTTP API 路由

```go
// router.go Line 392, 408
// 重新派发接口调用 RedispatchRequirement
```

**分析：** 前端通过 API 触发重新派发，这是人工操作，可以接受。

---

## 五、问题总结

### 🔴 严重问题（必须修复）

1. **`claudecode/processor.go:1101`** - Claude Code 完成后自动标记 completed
   - 影响：完全绕过状态机，AI 无法根据工作结果判断状态
   - 建议：移除自动标记，让 AI 通过 CLI 命令执行状态转换

2. **`requirement_dispatch_service.go`** - 派发流程硬编码状态流转
   - 影响：preparing → coding 的转换是自动的，不是通过状态机
   - 建议：派发成功后保持 preparing 状态，让 AI 自行决定何时进入 coding

3. **`requirement_service.go:154`** - PR webhook 直接标记 pr_opened
   - 影响：绕过状态机
   - 建议：PR webhook 触发后，通知 AI，由 AI 决定状态转换

### 🟡 中等问题（需要评估）

4. **错误处理中的 `MarkFailed()`**
   - 多处错误处理直接标记 failed
   - 需要区分：系统错误 vs 业务失败
   - 系统错误可以保留自动标记
   - 业务失败应该由 AI 判断

### 🟢 可以保留

5. **`RedispatchRequirement()`** - 人工重新派发
   - 这是管理操作，人工触发，可以保留

6. **`heartbeat_scheduler.go:143`** - 清理过期需求
   - 系统维护操作，可以保留

---

## 六、建议重构方案

### 目标架构

```
┌─────────────────┐
│   AI Agent      │
│  (Claude Code)  │
└────────┬────────┘
         │
         │ 1. 执行任务
         │ 2. 判断结果
         │ 3. 执行 CLI 命令
         ▼
┌─────────────────┐     ┌─────────────────┐
│  taskmanager    │────▶│   StateMachine  │
│   statemachine  │     │   (状态机引擎)   │
│   execute       │     └────────┬────────┘
└─────────────────┘              │
                                 │ 触发 Hook
                                 ▼
┌─────────────────┐     ┌─────────────────┐
│  taskmanager    │◀────│   Transition    │
│ requirement     │     │   Hook          │
│ update-state    │     │ (可选：调用API)  │
└────────┬────────┘     └─────────────────┘
         │
         │ 更新数据库
         ▼
┌─────────────────┐
│   Requirement   │
│   (数据库)       │
└─────────────────┘
```

### 具体修改清单

#### 1. 移除自动状态流转

**文件：`infrastructure/claudecode/processor.go`**
- 删除 `if success { requirement.MarkCompleted() ... }` 代码块
- 保留 Hook 触发、Token 统计、分身清理逻辑
- 添加日志提示 AI 需要手动执行状态转换

**文件：`application/requirement_dispatch_service.go`**
- 移除 `StartDispatch()` 中的 `requirement.MarkCoding()` 调用
- 派发成功后状态保持为 `preparing`，由 AI 自行决定何时进入 `coding`

**文件：`application/requirement_service.go`**
- 移除 `ReportRequirementPROpened()` 中的 `MarkPROpened()` 调用
- 改为触发 Hook 通知 AI，由 AI 执行状态转换

#### 2. 完善状态机 Hook

- 在关键转换点添加 Hook 配置（如 preparing→coding, coding→completed）
- Hook 可以通过 webhook 通知外部系统，或直接执行命令

#### 3. AI 提示词更新

确保 AI 知道如何执行状态转换：

```
【状态管理工作流】

1. 查看当前状态和可用触发器：
   taskmanager requirement get-state --id ${REQUIREMENT_ID}

2. 根据工作结果选择触发器执行转换：
   taskmanager statemachine execute --machine=${STATE_MACHINE_NAME} --from=<当前状态> --trigger=<触发器>

3. 同步状态到需求：
   taskmanager requirement update-state --id ${REQUIREMENT_ID} --status <新状态>

注意：系统不会自动更新需求状态，你必须手动执行上述命令。
```

---

## 七、风险与注意事项

1. **向后兼容性**：如果已有需求依赖自动状态流转，需要评估影响
2. **AI 提示**：确保 AI 明确知道需要手动管理状态
3. **错误处理**：系统错误（如网络故障）vs 业务错误（如代码编译失败）需要区分
4. **监控告警**：状态长时间不更新时需要告警

---

## 八、相关文件清单

| 文件路径 | 修改类型 | 优先级 |
|---------|---------|--------|
| `backend/infrastructure/claudecode/processor.go` | 移除 MarkCompleted | 🔴 P0 |
| `backend/application/requirement_dispatch_service.go` | 移除 MarkCoding | 🔴 P0 |
| `backend/application/requirement_service.go` | 移除 MarkPROpened | 🔴 P0 |
| `backend/domain/requirement.go` | 保留方法（供 CLI 使用）| 🟡 P1 |
| `backend/application/heartbeat_scheduler.go` | 保留（系统维护）| 🟢 P2 |
