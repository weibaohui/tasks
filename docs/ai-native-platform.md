# AI 原生软件开发平台

## 核心洞察

**这是一个专门为 AI Agent 打造的软件开发平台。**

整个系统的设计理念是：让人类成为"监督者"，而 AI Agent 成为真正的"执行者"。系统不仅记录和管理软件开发过程中的需求、任务、状态机，更是为 AI Agent 提供了完整的执行环境和工具链。

---

## 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                    AI 原生开发平台                           │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐   │
│  │   人类       │    │   AI 调度器   │    │   AI Agent  │   │
│  │  (监督者)    │───→│  (编排层)    │───→│   (执行者)   │   │
│  └─────────────┘    └─────────────┘    └─────────────┘   │
│         ↑                   │                   │          │
│         │                   ↓                   │          │
│         │           ┌─────────────┐            │          │
│         └───────────│  状态机引擎  │◀───────────┘          │
│                     └─────────────┘                       │
│                            │                               │
│                            ↓                               │
│                     ┌─────────────┐                       │
│                     │   Hook 系统   │                       │
│                     │  (扩展点)     │                       │
│                     └─────────────┘                       │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 核心能力

### 1. 状态机编排

状态机是系统的核心编排机制：

- **状态定义**：为项目定义业务流程状态（如：待开发 → 开发中 → 测试中 → 已完成）
- **转换触发**：支持 webhook 和 command 类型 hook，AI 可通过调用触发状态转换
- **上下文注入**：Hook 执行时自动注入 `requirement_id`、`project_id`、`from_state`、`to_state`、`trigger` 等上下文

```go
// Hook 上下文结构
type HookContext struct {
    RequirementID   string  // 需求 ID
    ProjectID       string  // 项目 ID
    StateMachineID  string  // 状态机 ID
    FromState       string  // 源状态
    ToState         string  // 目标状态
    Trigger         string  // 触发器
    HookName        string  // Hook 名称
    HookType        string  // Hook 类型 (webhook/command)
}
```

### 2. CLI 命令体系

AI Agent 通过 CLI 与系统交互：

| 命令 | 功能 |
|------|------|
| `taskmanager statemachine create` | 创建状态机 |
| `taskmanager statemachine list` | 列出状态机 |
| `taskmanager statemachine transition` | 触发状态转换 |
| `taskmanager statemachine state` | 获取当前状态 |
| `taskmanager statemachine history` | 获取转换历史 |
| `taskmanager statemachine bind` | 绑定类型到状态机 |
| `taskmanager state` | 获取项目状态汇总 |
| `taskmanager hook list` | 列出 Hook 配置 |
| `taskmanager requirement delete` | 删除需求 |

### 3. Command 类型 Hook

Hook 不仅能调用 webhook，还能执行二进制命令：

```json
{
  "type": "command",
  "config": {
    "command": "echo 'Transition: {{from_state}} → {{to_state}}'",
    "timeout": 30
  }
}
```

支持模板变量替换，AI 可以在命令中引用上下文信息。

---

## AI 工作流

### 场景：AI 自主完成一个需求

```
1. AI 接收任务
   └── "实现用户登录功能"

2. AI 创建需求
   └── POST /api/v1/requirements
   └── title: "用户登录功能"

3. AI 触发状态转换
   └── taskmanager statemachine transition <req-id> -t start

4. AI 执行开发
   └── 编写代码、运行测试

5. AI 提交产物
   └── 触发 complete 转换，Hook 自动记录

6. AI 发起 Code Review
   └── 触发 review 转换，通知相关人
```

### 场景：人类监督 AI 开发

```
1. 人类下达指令
   └── "完成用户认证模块"

2. AI 调度器接收
   └── 分析任务、创建工作目录

3. AI Agent 执行
   └── 在独立 workspace 中开发

4. 状态自动更新
   └── Hook 记录每个转换节点

5. 人类接收通知
   └── PR 创建、Review 请求

6. 人类验收
   └── approve/reject
```

---

## 与传统开发平台对比

| 维度 | 传统平台 | AI 原生平台 |
|------|---------|-------------|
| 执行主体 | 人类 | AI Agent |
| 任务拆解 | 人工 | AI 自主 |
| 进度更新 | 人工 | AI 自动 |
| 状态流转 | 人工触发 | AI/规则驱动 |
| 通知方式 | 人工发送 | Hook 自动 |
| 干预时机 | 全程 | 关键节点 |

---

## 系统边界

### AI 负责

- 需求理解和拆解
- 代码编写和测试
- 状态转换触发
- 进度实时更新
- 产物提交和管理

### 人类负责

- 需求最终确认
- Code Review
- 部署审批
- 异常干预

---

## 设计原则

1. **AI-First**：所有功能设计优先考虑 AI 的使用场景
2. **文本化交互**：CLI 和 Hook 是 AI 与系统交互的主要方式
3. **可编程**：系统所有功能可通过 API/CLI 调用
4. **状态可见**：所有状态变化通过 Hook 记录，AI 可查询完整历史
5. **自主执行**：AI 可在独立环境中执行，无需人工干预

---

## 扩展点

### Hook 系统

状态转换时自动执行扩展操作：

- **Webhook**：调用外部服务
- **Command**：执行本地命令
- **模板变量**：自动注入上下文信息

### 状态机配置

每个项目可配置独立的状态机：

- 自定义状态定义
- 自定义转换规则
- 自定义 Hook
- 多类型需求支持

---

## 下一步演进

1. **OODA 调度器**：实现 AI 调度器，自动观察-分析-决策-行动循环
2. **Agent 分身**：AI 自动创建工作副本，并行处理多个需求
3. **自动恢复**：Agent 崩溃后自动从上次状态恢复
4. **智能通知**：基于上下文智能选择通知时机和方式
