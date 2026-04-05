# StateMachine CLI 使用指南

## 概述

StateMachine CLI 是一个**纯通用、无业务耦合**的状态机规则引擎。它只负责状态流转规则的定义、校验和转换计算，不管理具体业务数据或实例ID。

**核心原则：**
- 不管理业务实例（如需求ID、订单号等）
- 只定义状态机模板（状态 + 触发器 + 流转规则）
- 业务层自行管理实例和状态存储

---

## 命令列表

| 命令 | 作用 | 核心场景 |
|------|------|----------|
| `list` | 列出所有状态机模板 | 查看系统支持的状态机 |
| `get` | 获取状态机模板完整规则 | 查看状态定义和流转规则 |
| `triggers` | 查询指定状态的可用触发器 | 获取当前状态可执行的操作 |
| `validate` | 验证从A到B的状态转换是否允许 | 预校验转换合法性 |
| `execute` | 执行状态转换，返回目标状态 | 通用转换计算（无存储） |

---

## 全局参数

所有命令都支持以下全局参数：

- `--machine, -m`: 状态机模板名称（必填）
- `--from, -f`: 源状态ID（triggers/execute 命令必填）
- `--to, -t`: 目标状态ID（validate 命令必填）
- `--trigger, -t`: 触发器名称（execute 命令必填）

---

## 逐命令详解

### 1. list - 列出所有状态机模板

**用途：** 查看系统中所有可用的通用状态机模板定义。

**命令：**
```bash
taskmanager statemachine list
```

**输出示例：**
```json
[
  {
    "name": "心跳任务流程",
    "description": "适用于心跳任务的简单流程：待处理 → 已完成",
    "initial_state": "todo",
    "states_count": 2,
    "transitions_count": 1
  },
  {
    "name": "完整开发流程",
    "description": "完整的软件开发流程",
    "initial_state": "code_commit",
    "states_count": 7,
    "transitions_count": 10
  }
]
```

---

### 2. get - 获取状态机模板详情

**用途：** 获取指定状态机模板的完整规则定义（状态列表 + 流转规则）。

**命令：**
```bash
taskmanager statemachine get --machine=<状态机名称>
# 或简写
taskmanager statemachine get -m <状态机名称>
```

**示例：**
```bash
taskmanager statemachine get --machine="心跳任务流程"
```

**输出示例：**
```json
{
  "name": "心跳任务流程",
  "description": "适用于心跳任务的简单流程：待处理 → 已完成",
  "initial_state": "todo",
  "states": [
    {"id": "todo", "name": "待处理", "is_final": false},
    {"id": "completed", "name": "已完成", "is_final": true}
  ],
  "transitions": [
    {
      "from": "todo",
      "to": "completed",
      "trigger": "complete",
      "description": "心跳任务完成"
    }
  ]
}
```

---

### 3. triggers - 查询指定状态的可用触发器

**用途：** 输入状态机名称和当前状态，返回该状态下所有可用的触发器列表。

**命令：**
```bash
taskmanager statemachine triggers --machine=<名称> --from=<状态>
# 或简写
taskmanager statemachine triggers -m <名称> -f <状态>
```

**示例：**
```bash
taskmanager statemachine triggers --machine="心跳任务流程" --from=todo
```

**输出示例：**
```json
{
  "machine": "心跳任务流程",
  "current_state": "todo",
  "triggers": [
    {
      "trigger": "complete",
      "to_state": "completed",
      "description": "心跳任务完成"
    }
  ],
  "count": 1
}
```

---

### 4. validate - 验证状态转换是否允许

**用途：** 验证从指定源状态到目标状态是否存在有效的转换规则。返回所有可达路径的触发器列表。

**命令：**
```bash
taskmanager statemachine validate --machine=<名称> --from=<源状态> --to=<目标状态>
# 或简写
taskmanager statemachine validate -m <名称> -f <源状态> -t <目标状态>
```

**示例 - 有效转换：**
```bash
taskmanager statemachine validate --machine="心跳任务流程" --from=todo --to=completed
```

**输出：**
```json
{
  "machine": "心跳任务流程",
  "from": "todo",
  "to": "completed",
  "from_state_exists": true,
  "to_state_exists": true,
  "valid": true,
  "transitions": [
    {"trigger": "complete", "description": "心跳任务完成"}
  ],
  "transitions_count": 1,
  "message": "从 'todo' 到 'completed' 有 1 个可用转换"
}
```

**示例 - 无效转换：**
```bash
taskmanager statemachine validate --machine="心跳任务流程" --from=completed --to=todo
```

**输出：**
```json
{
  "machine": "心跳任务流程",
  "from": "completed",
  "to": "todo",
  "from_state_exists": true,
  "to_state_exists": true,
  "valid": false,
  "transitions": null,
  "transitions_count": 0,
  "message": "从 'completed' 到 'todo' 没有直接的转换规则"
}
```

---

### 5. execute - 执行通用状态转换

**用途：** 执行状态转换计算：输入状态机模板、当前状态、触发器，返回目标状态。

**特点：**
- 纯计算、无存储、无业务ID
- 不管理业务实例，只负责规则计算
- 业务层自行保存转换结果

**命令：**
```bash
taskmanager statemachine execute --machine=<名称> --from=<状态> --trigger=<触发器>
# 或简写
taskmanager statemachine execute -m <名称> -f <状态> -t <触发器>
```

**示例 - 成功：**
```bash
taskmanager statemachine execute --machine="心跳任务流程" --from=todo --trigger=complete
```

**输出：**
```json
{
  "success": true,
  "machine": "心跳任务流程",
  "from_state": "todo",
  "to_state": "completed",
  "trigger": "complete",
  "description": "心跳任务完成"
}
```

**示例 - 失败（触发器不存在）：**
```bash
taskmanager statemachine execute --machine="心跳任务流程" --from=todo --trigger=invalid
```

**输出：**
```json
{
  "error": true,
  "message": "状态 'todo' 不支持触发器 'invalid'",
  "machine": "心跳任务流程",
  "state": "todo",
  "trigger": "invalid",
  "available_triggers": ["complete"]
}
```

---

## 典型工作流

### 场景：研发发布流程

```bash
# 1. 查看可用的状态机
taskmanager statemachine list

# 2. 获取完整开发流程的规则
taskmanager statemachine get --machine="完整开发流程"

# 3. 查看当前状态（code_commit）有哪些可用触发器
taskmanager statemachine triggers -m "完整开发流程" -f code_commit
# 输出: { "triggers": [{ "trigger": "submit_review", "to_state": "code_review" }] }

# 4. 验证转换是否合法
taskmanager statemachine validate -m "完整开发流程" -f code_commit -t code_review
# 输出: { "valid": true }

# 5. 执行转换（业务层自行保存结果）
RESULT=$(taskmanager statemachine execute -m "完整开发流程" -f code_commit -t submit_review)
echo "转换结果: $RESULT"
# 输出: { "success": true, "to_state": "code_review" }
```

---

## 业务层集成方式

### 解耦设计

状态机 CLI 只提供**规则计算**，业务层负责**数据存储**：

```bash
# 业务层工作流示例

# 1. 业务层持有：业务ID + 当前状态
BUSINESS_ID="order-123"
CURRENT_STATUS="pending"

# 2. 调用状态机 CLI 做规则校验/转换
NEW_STATUS=$(taskmanager statemachine execute \
  --machine="order-workflow" \
  --from="$CURRENT_STATUS" \
  --trigger="pay" \
  | jq -r '.to_state')

# 3. 业务层自己保存新状态
update_order_status "$BUSINESS_ID" "$NEW_STATUS"
```

### 与需求系统的关联

虽然状态机 CLI 是纯通用的，但在本系统中通常与需求（Requirement）关联使用：

```bash
# 1. 获取需求当前状态（通过 API 或其他 CLI）
REQUIREMENT_ID="req-abc-123"

# 2. 查看该状态下可用的触发器
# 假设当前需求状态是 "todo"
taskmanager statemachine triggers \
  --machine="心跳任务流程" \
  --from=todo

# 3. 执行状态转换
taskmanager statemachine execute \
  --machine="心跳任务流程" \
  --from=todo \
  --trigger=complete

# 4. 业务层将新状态保存到需求记录
```

---

## 参数速查表

| 参数 | 简写 | 适用于 | 说明 |
|------|------|--------|------|
| `--machine` | `-m` | 所有命令 | 状态机模板名称 |
| `--from` | `-f` | triggers, execute, validate | 源状态（当前状态） |
| `--to` | `-t` | validate | 目标状态 |
| `--trigger` | `-t` | execute | 触发器名称 |

---

## 输出格式

所有命令输出**紧凑 JSON**格式，便于脚本处理：

```bash
# 直接获取目标状态
TO_STATE=$(taskmanager statemachine execute -m "workflow" -f todo -t complete | jq -r '.to_state')
echo $TO_STATE  # 输出: completed

# 检查转换是否合法
if taskmanager statemachine validate -m "workflow" -f todo -t done | jq -e '.valid' > /dev/null; then
  echo "转换合法"
else
  echo "转换不合法"
fi
```

---

## 相关文档

- [DDD.md](./DDD.md) - DDD 架构约束与最佳实践
- [development-workflow.md](./development-workflow.md) - 开发流程、测试规范
- [StateMachine Management UI](../frontend/src/components/StateMachineManagement/) - 状态机管理界面
