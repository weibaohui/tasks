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
