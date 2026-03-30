# Requirement 状态变更 Hook 系统需求文档

## 1. 项目概述

### 1.1 项目名称
**Requirement State Change Hook System** - 需求状态变更 Hook 系统

### 1.2 项目目标
为 Requirement（需求）领域模型添加状态变更钩子机制，实现：
1. **Hook 可配置化** - 通过数据库配置每个 Hook 点的动作列表
2. **动作可触发新分身** - 每个动作可以触发另一个 Coding Agent 分身
3. **分身自动销毁** - Replica Agent 用完即销毁，做成代码约束而非 Hook

### 1.3 背景与动机

#### 问题 1：Hook 不可配置
当前 Hook 是硬编码的，无法通过配置新增 Hook 或修改行为。需要：
- 数据库配置表
- 前端管理界面
- 每个 Hook 点可挂载多个动作

#### 问题 2：分身未自动销毁
当前代码中分身 Agent 创建后从未删除。需要：
- 代码约束：分身用完即销毁
- 销毁时机：PR 创建成功 或 任务失败

## 2. 功能需求

### 2.1 Hook 配置化

#### 2.1.1 Hook 配置表设计

```sql
CREATE TABLE requirement_hook_configs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,                    -- Hook 配置名称
    trigger_point TEXT NOT NULL,           -- 触发点: start_dispatch, mark_coding, mark_failed, mark_pro_opened
    action_type TEXT NOT NULL,             -- 动作类型: trigger_agent, notification, webhook
    action_config TEXT NOT NULL,          -- JSON配置: agent_id, prompt_template, timeout等
    enabled INTEGER DEFAULT 1,             -- 是否启用
    priority INTEGER DEFAULT 50,           -- 执行优先级
    created_at TEXT,
    updated_at TEXT
);

CREATE TABLE requirement_hook_action_logs (
    id TEXT PRIMARY KEY,
    hook_config_id TEXT NOT NULL,
    requirement_id TEXT NOT NULL,
    action_type TEXT NOT NULL,
    status TEXT NOT NULL,                 -- pending, running, success, failed
    result TEXT,                          -- 动作执行结果
    error TEXT,                           -- 错误信息
    started_at TEXT,
    completed_at TEXT,
    FOREIGN KEY (hook_config_id) REFERENCES requirement_hook_configs(id)
);
```

#### 2.1.2 支持的动作类型

| 动作类型 | 配置参数 | 说明 |
|---------|---------|------|
| `trigger_agent` | agent_id, prompt_template, timeout | 触发另一个 Agent 执行任务 |
| `notification` | channel, template | 发送通知 |
| `webhook` | url, method, headers, body_template | 发送 Webhook |

#### 2.1.3 trigger_agent 动作配置

```json
{
    "agent_id": "agt-xxx",           // 指定要触发的 Agent
    "prompt_template": "根据上下文完成: ${requirement.title}",  // 支持变量替换
    "timeout_minutes": 30,           // 超时时间
    "workspace_template": "/tmp/${requirement.id}",  // 工作目录模板
    "context": {
        "include_project": true,      // 是否包含项目信息
        "include_requirement": true, // 是否包含需求信息
        "include_history": false      // 是否包含对话历史
    }
}
```

### 2.2 分身自动销毁

#### 2.2.1 销毁时机

| 时机 | 触发条件 | 销毁行为 |
|------|---------|---------|
| PR 创建成功 | `MarkPROpened` 调用时 | 销毁分身 + 清理 workspace |
| 任务失败 | `MarkFailed` 调用时 | 销毁分身 + 清理 workspace |
| 超时 | 后台定时检查 | 销毁超时分身 |

#### 2.2.2 代码约束

```go
// Replica Agent 必须在使用后销毁
type ReplicaAgentManager struct {
    agentRepo domain.AgentRepository
}

// Dispose 销毁分身（强制方法，不返回错误）
func (m *ReplicaAgentManager) Dispose(ctx context.Context, replicaAgentID string) {
    // 1. 删除 Agent 记录
    // 2. 清理 Agent 配置的工作目录
    // 3. 记录销毁日志
}

// EnsureDisposed 确保销毁（幂等方法）
func (m *ReplicaAgentManager) EnsureDisposed(ctx context.Context, replicaAgentID, workspacePath string) {
    if replicaAgentID == "" {
        return
    }
    m.Dispose(ctx, replicaAgentID)
    if workspacePath != "" {
        os.RemoveAll(workspacePath)
    }
}
```

### 2.3 前端界面

#### 2.3.1 Hook 配置管理页面

| 功能 | 说明 |
|------|------|
| 列表页 | 显示所有 Hook 配置，支持启用/禁用 |
| 新建页 | 选择触发点、动作类型、配置参数 |
| 编辑页 | 修改配置 |
| 日志页 | 查看动作执行历史 |

#### 2.3.2 页面结构

```
/hooks
├── /hooks/configs              # Hook 配置列表
├── /hooks/configs/new         # 新建配置
├── /hooks/configs/:id/edit    # 编辑配置
└── /hooks/logs                # 执行日志
```

## 3. 状态变更事件

### 3.1 触发点定义

| 触发点 | 值 | 触发时机 | 可用上下文变量 |
|--------|---|---------|--------------|
| `start_dispatch` | `start_dispatch` | `StartDispatch()` | requirement, agent |
| `mark_coding` | `mark_coding` | `MarkCoding()` | requirement, agent, workspace |
| `mark_failed` | `mark_failed` | `MarkFailed()` | requirement, error |
| `mark_pr_opened` | `mark_pr_opened` | `MarkPROpened()` | requirement, pr_url |

### 3.2 上下文变量

| 变量 | 说明 | 示例 |
|------|------|------|
| `${requirement.id}` | 需求 ID | `req-001` |
| `${requirement.title}` | 需求标题 | `登录功能` |
| `${requirement.description}` | 需求描述 | `实现用户登录` |
| `${requirement.acceptance_criteria}` | 验收标准 | `能正常登录` |
| `${project.id}` | 项目 ID | `proj-001` |
| `${project.name}` | 项目名称 | `电商系统` |
| `${project.git_repo_url}` | 仓库地址 | `git@xxx` |
| `${agent.id}` | 分身 Agent ID | `agt-xxx` |
| `${workspace.path}` | 工作目录 | `/tmp/xxx` |

## 4. 验收标准

### 4.1 Hook 配置化验收

- [ ] 数据库表正确创建
- [ ] CRUD 接口完整
- [ ] 前端界面可配置
- [ ] 动作可正确执行

### 4.2 分身自动销毁验收

- [ ] `MarkPROpened` 时自动销毁分身
- [ ] `MarkFailed` 时自动销毁分身
- [ ] 销毁是强制行为，不可跳过
- [ ] 销毁后分身记录从数据库删除

### 4.3 触发 Agent 动作验收

- [ ] 配置 prompt 模板正确渲染
- [ ] 上下文变量正确替换
- [ ] 新分身正确创建并执行
- [ ] 执行结果正确记录

## 5. 术语表

| 术语 | 定义 |
|------|------|
| Hook 配置 | 存储在数据库中的可配置 Hook 定义 |
| 触发点 | 状态变更的方法名 |
| 动作 | Hook 配置中定义的具体执行行为 |
| 上下文 | 触发动作时可用的变量数据 |
| Replica Agent | 分身 Agent，用完即销毁 |

## 6. 参考文档

- `06-requirement-state-hook-design.md` - 详细设计文档
