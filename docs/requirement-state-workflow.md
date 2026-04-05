# Requirement 状态管理工作流

## 概述

本工作流描述如何使用 `taskmanager requirement` 和 `taskmanager statemachine` 命令配合，实现需求状态的获取、状态机转换和状态更新。

## 命令说明

### 1. get-state - 获取需求当前状态

获取需求在状态机中的当前状态，以及需求的 status 字段。

```bash
taskmanager requirement get-state --id <requirement-id>
```

**输出示例：**
```json
{
  "requirement_id": "req-abc-123",
  "state_machine_id": "sm-xyz-456",
  "current_state": "todo",
  "state_name": "待处理",
  "requirement_status": "todo"
}
```

**错误情况（未初始化状态机）：**
```json
{
  "error": true,
  "requirement_id": "req-abc-123",
  "message": "获取需求状态失败: not found"
}
```

### 2. update-state - 更新需求状态

更新需求的 status 字段。通常用于将状态机执行结果同步到需求状态。

```bash
taskmanager requirement update-state --id <requirement-id> --status <new-status>
```

**输出示例：**
```json
{
  "success": true,
  "requirement_id": "req-abc-123",
  "old_status": "todo",
  "new_status": "completed",
  "message": "状态已从 'todo' 更新为 'completed'"
}
```

## 完整工作流示例

### 场景：需求状态流转

假设有一个需求使用 "心跳任务流程" 状态机，当前状态为 "todo"，需要执行 "complete" 触发器转换到 "completed" 状态。

#### Step 1: 获取需求当前状态

```bash
taskmanager requirement get-state --id req-abc-123
```

输出：
```json
{
  "requirement_id": "req-abc-123",
  "current_state": "todo",
  "requirement_status": "todo"
}
```

#### Step 2: 查询可用触发器

```bash
taskmanager statemachine triggers --machine="心跳任务流程" --from=todo
```

输出：
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

#### Step 3: 验证转换是否合法

```bash
taskmanager statemachine validate --machine="心跳任务流程" --from=todo --to=completed
```

输出：
```json
{
  "valid": true,
  "transitions": [{"trigger": "complete"}]
}
```

#### Step 4: 执行状态转换

```bash
taskmanager statemachine execute --machine="心跳任务流程" --from=todo --trigger=complete
```

输出：
```json
{
  "success": true,
  "from_state": "todo",
  "to_state": "completed",
  "trigger": "complete"
}
```

#### Step 5: 更新需求状态

将状态机执行结果同步到需求：

```bash
taskmanager requirement update-state --id req-abc-123 --status completed
```

输出：
```json
{
  "success": true,
  "requirement_id": "req-abc-123",
  "old_status": "todo",
  "new_status": "completed",
  "message": "状态已从 'todo' 更新为 'completed'"
}
```

## 自动化脚本示例

```bash
#!/bin/bash

REQ_ID="req-abc-123"
MACHINE="心跳任务流程"
TRIGGER="complete"

echo "=== Step 1: 获取需求当前状态 ==="
STATE_INFO=$(taskmanager requirement get-state --id "$REQ_ID")
CURRENT_STATE=$(echo "$STATE_INFO" | jq -r '.current_state')
echo "当前状态: $CURRENT_STATE"

echo "=== Step 2: 执行状态转换 ==="
RESULT=$(taskmanager statemachine execute --machine="$MACHINE" --from="$CURRENT_STATE" --trigger="$TRIGGER")
SUCCESS=$(echo "$RESULT" | jq -r '.success')

if [ "$SUCCESS" != "true" ]; then
  echo "状态转换失败"
  echo "$RESULT"
  exit 1
fi

NEW_STATE=$(echo "$RESULT" | jq -r '.to_state')
echo "转换成功，新状态: $NEW_STATE"

echo "=== Step 3: 更新需求状态 ==="
UPDATE_RESULT=$(taskmanager requirement update-state --id "$REQ_ID" --status "$NEW_STATE")
echo "$UPDATE_RESULT"
```

## 命令速查

| 命令 | 用途 | 示例 |
|------|------|------|
| `requirement get-state` | 获取需求当前状态 | `taskmanager requirement get-state -i req-123` |
| `requirement update-state` | 更新需求状态 | `taskmanager requirement update-state -i req-123 -s completed` |
| `statemachine triggers` | 查询可用触发器 | `taskmanager statemachine triggers -m "workflow" -f todo` |
| `statemachine validate` | 验证转换合法性 | `taskmanager statemachine validate -m "workflow" -f todo -t completed` |
| `statemachine execute` | 执行状态转换 | `taskmanager statemachine execute -m "workflow" -f todo -t complete` |

## 注意事项

1. **状态机与需求状态的区分**
   - `state_machine` 命令只处理状态机规则，不涉及业务数据
   - `requirement` 命令处理具体的需求业务数据
   - 状态机执行结果需要通过 `update-state` 同步到需求

2. **状态一致性**
   - 建议在状态机转换成功后立即更新需求状态
   - 可以使用脚本自动化整个过程

3. **错误处理**
   - 如果需求未初始化状态机，`get-state` 会返回错误
   - 需要先使用 HTTP API 初始化需求状态机
